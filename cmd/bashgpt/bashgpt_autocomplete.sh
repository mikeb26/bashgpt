#!/bin/bash

# "--" is used to delineate between the user's prompt, and the reponse from OpenAI's GPT API
# after the user pressed the [TAB] key. this allows the user to simply press enter after viewing
# the response in order to execute the returned command(s). Additionally, this also allows both
# the prompt along with the response to be stored in .bash_history for repeatability.
#
# see main.go:argsToPromptAndCmd() before making changes here
_bashgpt_completion() {
    cur="${COMP_WORDS[COMP_CWORD]}"
    COMPREPLY=( "$cur -- $(bashgpt sh ${COMP_WORDS[@]})" )
    return 0
}

alias ?='bashgpt sh'
complete -F _bashgpt_completion '?'
