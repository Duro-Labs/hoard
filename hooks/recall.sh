#!/usr/bin/env bash
set -u
[ "${BASH_VERSINFO[0]}" -ge 4 ] || exit 0
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_lib.sh"
hoard_load_config
hoard_disabled recall && exit 0
[ -n "$HOARD_VAULT_ROOT" ] && [ -d "$HOARD_VAULT_ROOT" ] || exit 0


command -v jq >/dev/null 2>&1 || exit 0
command -v rg >/dev/null 2>&1 || exit 0

scope=$HOARD_SCOPE
base=$HOARD_VAULT_ROOT
roots=("$base/Agents/$scope")
[ "$scope" != "$HOARD_DEFAULT_SCOPE" ] && roots+=("$base/Agents/$HOARD_DEFAULT_SCOPE")
for r in "$base"/*/; do r="${r%/}"; [ "$r" != "$base/Agents" ] && roots+=("$r"); done
present=(); for r in "${roots[@]}"; do [ -d "$r" ] && present+=("$r"); done
[ "${#present[@]}" -eq 0 ] && exit 0
roots=("${present[@]}")
root_of() { local r; for r in "${roots[@]}"; do [[ $1 == "$r"/* ]] && { printf '%s' "$r"; return; }; done; }
subject_of() {
  local r rel; r=$(root_of "$1"); rel="${1#"$r"/}"
  if [ "$r" = "$base" ]; then printf '%s' "${rel%%/*}"; else printf '%s/%s' "${r#"$base"/}" "${rel%%/*}"; fi
}
state_dir="$HOARD_STATE/recall"

hoard_read_payload
[ -z "$HOARD_PAYLOAD" ] && exit 0
prompt=$(hoard_prompt)
[ "${#prompt}" -lt 3 ] && exit 0
cwd=$(hoard_cwd)
session=$(hoard_session)
prompt="${prompt:0:4000}"

is_issue() { [[ "$1" =~ ^[0-9]{3,5}$ ]] && ! [[ "$1" =~ ^20[0-9]{2}$ ]]; }

stop_re='^(this|that|these|those|with|from|what|when|where|which|while|have|has|will|would|should|could|about|into|just|like|also|then|than|them|they|their|there|here|does|done|make|sure|need|want|please|check|look|take|thing|things|something|anything|issue|issues|problem|still|again|some|more|most|very|really|because|before|after|over|under|only|even|much|many|each|other|another|being|were|been|your|yours|mine|ours|claude|vault|related|prompt|going|know|think|help|show|tell|give|find|okay|right|well|work|working|works|worked|doing|done|able|want|wants|wanted|kind|sort|ones|through|between|within|without|around|since|until|maybe|yeah|yes|also|both|same|different|good|great|nice|first|second|last|next|previous|current|latest|already|actually|basically|probably|possibly|instead|rather|whether|either|neither|every|anyway|however|though|although|unless|always|never|often|sometimes|somewhere|anywhere|everything|nothing|someone|anyone|everyone|itself|himself|herself|themselves|myself|yourself|ourselves)$'

declare -a terms=() issues=()
declare -A seen=() kind=()
add_term() {
  local t="$1"
  [ -z "$t" ] && return
  [ -n "${seen[$t]:-}" ] && return
  [ "${#terms[@]}" -ge 12 ] && return
  seen[$t]=1; kind[$t]="$2"; terms+=("$t")
}

while IFS= read -r n; do
  [ "${#issues[@]}" -ge 5 ] && break
  is_issue "$n" || continue
  [ -n "${seen[$n]:-}" ] && continue
  add_term "$n" issue; issues+=("$n")
done < <(printf '%s' "$prompt" | grep -oE '\b[0-9]{3,5}\b')
if [ -n "$cwd" ] && [ -d "$cwd" ]; then
  branch=$(git -C "$cwd" symbolic-ref --short -q HEAD 2>/dev/null || true)
  n=$(printf '%s' "$branch" | grep -oE '[0-9]{3,5}' | while IFS= read -r x; do is_issue "$x" && { echo "$x"; break; }; done)
  [ -n "$n" ] && { [ -z "${seen[$n]:-}" ] && issues+=("$n"); add_term "$n" issue; }
fi
while IFS= read -r q; do add_term "$q" phrase; done \
  < <(printf '%s' "$prompt" | grep -oE '`[^`]{3,60}`|"[^"]{3,60}"' | sed -E 's/^[`"](.*)[`"]$/\1/')
while IFS= read -r q; do add_term "$q" ident; done \
  < <(printf '%s' "$prompt" | grep -oE '\b[A-Za-z][A-Za-z0-9]*[._][A-Za-z0-9._]+\b|\b[A-Z][a-z]+[A-Z][A-Za-z0-9]+\b' | sort -u)
while IFS= read -r w; do
  printf '%s\n' "$w" | grep -qE "$stop_re" && continue
  frag=0
  for t in "${terms[@]}"; do
    case "${kind[$t]}" in phrase|ident) ;; *) continue;; esac
    lt="${t,,}"; [[ "$lt" != "$w" && "$lt" == *"$w"* ]] && { frag=1; break; }
  done
  [ "$frag" -eq 1 ] || add_term "$w" word
done < <(printf '%s' "$prompt" | tr 'A-Z' 'a-z' | grep -oE '\b[a-z]{4,}\b' | sort -u)

[ "${#terms[@]}" -eq 0 ] && exit 0

declare -A hit=() cand=() score=() matched=()
for t in "${terms[@]}"; do
  rgw=(); [ "${kind[$t]}" = issue ] && rgw=(-w)
  while IFS= read -r f; do hit["$t|$f"]=1; cand["$f"]=1; done \
    < <(rg -ilF "${rgw[@]}" --glob '*.md' --glob '*.html' --glob '!**/_Inbox/scratch/**' --glob '!**/_Archive/**' -- "$t" "${roots[@]}" 2>/dev/null)
done
for n in "${issues[@]}"; do
  while IFS= read -r f; do cand["$f"]=1; done \
    < <(rg --files --glob '*.md' --glob '*.html' --glob '!**/_Inbox/scratch/**' --glob '!**/_Archive/**' "${roots[@]}" 2>/dev/null | grep -F -- "$n")
done
[ "${#cand[@]}" -eq 0 ] && exit 0

for f in "${!cand[@]}"; do
  rel="${f#"$(root_of "$f")"/}"; subject="${rel%%/*}"; lsub="${subject,,}"; lrel="${rel,,}"
  s=0; m=""
  for t in "${terms[@]}"; do
    lt="${t,,}"
    if [ "${kind[$t]}" = issue ]; then
      if [[ "$lrel" =~ (^|[^0-9])$t([^0-9]|$) ]] || [ -n "${hit[$t|$f]:-}" ]; then s=$((s+5)); m+="$t, "; fi
    elif [[ "$lsub" == *"$lt"* ]]; then s=$((s+2)); m+="$t, "
    elif [ -n "${hit[$t|$f]:-}" ]; then
      case "${kind[$t]}" in phrase|ident) s=$((s+2));; *) s=$((s+1));; esac
      m+="$t, "
    fi
  done
  [ "$s" -ge 3 ] && { score["$f"]=$s; matched["$f"]="${m%, }"; }
done
[ "${#score[@]}" -eq 0 ] && exit 0

age_of() {
  local d=$(( ( $(date +%s) - $1 ) / 86400 ))
  if [ "$d" -le 0 ]; then echo today
  elif [ "$d" -lt 7 ]; then echo "${d}d ago"
  elif [ "$d" -lt 60 ]; then echo "$((d/7))w ago"
  else echo "$((d/30))mo ago"; fi
}

mkdir -p "$state_dir" 2>/dev/null || exit 0
find "$state_dir" -type f -mtime +7 -delete 2>/dev/null
state_file="$state_dir/$session"
touch "$state_file" 2>/dev/null

already_shown() { grep -qxF "$(printf '%s\t%s' "$1" "$2")" "$state_file" 2>/dev/null; }
mark_shown() {
  grep -vF "$(printf '%s\t' "$1")" "$state_file" 2>/dev/null > "$state_file.tmp"
  printf '%s\t%s\n' "$1" "$2" 2>/dev/null >> "$state_file.tmp"
  mv -f "$state_file.tmp" "$state_file" 2>/dev/null
}

declare -A subj_max=() subj_cnt=() subj_durable=() subj_drafts=()
for f in "${!score[@]}"; do
  rel="${f#"$(root_of "$f")"/}"; subject=$(subject_of "$f"); s=${score[$f]}
  [ "${subj_max[$subject]:-0}" -lt "$s" ] && subj_max[$subject]=$s
  subj_cnt[$subject]=$(( ${subj_cnt[$subject]:-0} + 1 ))
  line="$s"$'\t'"$(hoard_mtime "$f")"$'\t'"$f"$'\n'
  if [[ "$rel" == */Drafts/* ]]; then subj_drafts[$subject]+="$line"; else subj_durable[$subject]+="$line"; fi
done

rank_subjects() {
  local subject extra pick top newest
  for subject in "${!subj_max[@]}"; do
    extra=$(( ${subj_cnt[$subject]} - 1 )); [ "$extra" -gt 3 ] && extra=3
    pick="${subj_durable[$subject]:-${subj_drafts[$subject]:-}}"
    top=$(printf '%s' "$pick" | sort -t$'\t' -k1,1nr -k2,2nr | head -3)
    [ -z "$top" ] && continue
    newest=$(printf '%s\n' "$top" | cut -f2 | sort -nr | head -1)
    printf '%s\t%s\t%s\t%s\n' "$(( ${subj_max[$subject]} + extra ))" "$newest" "$subject" "$(printf '%s' "$top" | tr '\n' '\037')"
  done | sort -t$'\t' -k1,1nr -k2,2nr | head -3
}

emit_subjects() {
  local out="" subject files cnt flines fmt f rel name title noun
  while IFS=$'\t' read -r _ _ subject files; do
    cnt=${subj_cnt[$subject]:-1}; flines=""
    while IFS=$'\t' read -r _ fmt f; do
      [ -z "$f" ] && continue
      already_shown "$f" "$fmt" && continue
      mark_shown "$f" "$fmt"
      rel="${f#"$(root_of "$f")"/}"
      if [[ "$rel" == */* ]]; then name="${rel#*/}"; else name="$subject"; fi
      case $f in
      *.html) title=$(grep -o -m1 '<title>[^<]*</title>' "$f" 2>/dev/null | sed 's/<[^>]*>//g') ;;
      *) title=$(grep -m1 '^# ' "$f" 2>/dev/null | sed 's/^# //') ;;
      esac
      title="${title:-$(basename "$f")}"
      flines+="    · ${name} — \"${title}\" · $(age_of "$fmt") · matched ${matched[$f]:-}"$'\n'
    done < <(printf '%s\n' "$files" | tr '\037' '\n')
    [ -z "$flines" ] && continue
    if [[ "$subject" == *.md ]]; then
      out+="- ${flines#    · }"
    else
      noun=docs; [ "$cnt" -eq 1 ] && noun=doc
      out+="- ${subject}/ — ${cnt} ${noun} match; start with:"$'\n'"${flines}"
    fi
  done
  [ -z "$out" ] && return 0
  local summary; summary=$(printf '%s, ' "${terms[@]}"); summary="${summary%, }"
  local ctx="[hoard recall] Related subjects (matched: ${summary}):"$'\n'"${out}"
  ctx+="Open your reply with one line beginning \"Related notes:\" that names these subject folders (under ${base}). Read the listed notes that bear on the task before acting; say briefly if one turns out unrelated."
  hoard_emit_context "$ctx"
}

rank_subjects | emit_subjects
exit 0
