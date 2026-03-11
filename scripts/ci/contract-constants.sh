extract_const_raw() {
  name="$1"
  awk -v name="${name}" '
    $0 ~ "^[[:space:]]*" name "[[:space:]]*=" {
      line = $0
      sub(/^[[:space:]]*[A-Za-z0-9_]+[[:space:]]*=[[:space:]]*/, "", line)
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", line)
      print line
      exit
    }
  ' "${CONTRACT_FILE}"
}

resolve_const() {
  name="$1"
  seen="$2"

  case "${seen}" in
    *"
${name}
"*)
      echo "Cyclic constant reference while resolving ${name}" >&2
      exit 1
      ;;
  esac

  raw="$(extract_const_raw "${name}")"
  if [ -z "${raw}" ]; then
    echo ""
    return 0
  fi

  case "${raw}" in
    \"*\")
      printf '%s\n' "${raw}" | sed 's/^"//; s/"$//'
      ;;
    [A-Za-z_][A-Za-z0-9_]*)
      resolve_const "${raw}" "${seen}
${name}
"
      ;;
    *)
      echo ""
      ;;
  esac
}

extract_const() {
  name="$1"
  resolve_const "$name" ""
}
