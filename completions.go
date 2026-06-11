package main

const targetsHelper = `__creo_targets() {
	local fiat_file
	if [ -f "fiat" ]; then
		fiat_file="fiat"
	else
		for f in *.fiat; do
			if [ -f "$f" ]; then
				fiat_file="$f"
				break
			fi
		done
	fi
	if [ -n "$fiat_file" ]; then
		local targets
		targets=$(grep -E '^[a-zA-Z0-9._-]+:' "$fiat_file" | sed 's/:.*//' 2>/dev/null)
		COMPREPLY+=( $(compgen -W "$targets" -- "$cur") )
	fi
}`

const langsHelper = `__creo_langs() {
	COMPREPLY+=( $(compgen -W "go c cxx cpp rust oci" -- "$cur") )
}`

const completionFunc = `_creo() {
	COMPREPLY=()
	local cur prev
	_get_comp_words_by_ref cur prev

	if [ ${COMP_CWORD} -eq 1 ]; then
		if [[ ${cur} == -* ]]; then
			COMPREPLY=( $(compgen -W "${options}" -- $cur) )
			return 0
		fi

		__creo_targets
		return 0
	fi

	if [[ ${cur} == -* ]]; then
		case ${prev} in
		*)
			if [[ $(hasword ${prev} ${options}) == "1" ]]; then
				COMPREPLY=( $(compgen -W "${options}" -- $cur) )
				return 0
			fi
			;;
		esac
	fi

	if [[ $(hasword ${prev} ${options}) == "1" ]]; then
		case ${prev} in
		-i|--init)
			__creo_langs
			;;
		-f|--file)
			complete_files
			;;
		--graph)
			COMPREPLY=( $(compgen -W "tree dot svg" -- "$cur") )
			;;
		*)
			__creo_targets
			;;
		esac
		return 0
	fi

	__creo_targets
	return 0
}`
