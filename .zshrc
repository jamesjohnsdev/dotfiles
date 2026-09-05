# If you come from bash you might have to change your $PATH.
# export PATH=$HOME/bin:$HOME/.local/bin:/usr/local/bin:$PATH

# Environment
export BAT_THEME=ansi
export MANROFFOPT="-c"
export MANPAGER="sh -c 'col -bx | bat -l man -p'"
export SUDO_EDITOR="$EDITOR"
export OMARCHY_PATH=$HOME/.local/share/omarchy
export PATH=$OMARCHY_PATH/bin:$PATH:$HOME/.local/bin

# Completion
autoload -Uz compinit
compinit

# Tool initialization
if [[ ${TERM:-} != "dumb" ]] && command -v starship &> /dev/null; then eval "$(starship init zsh)"; fi
if command -v zoxide &> /dev/null; then eval "$(zoxide init zsh)"; fi
if command -v fzf &> /dev/null; then source <(fzf --zsh) 2>/dev/null; fi

# Omarchy fns (tdl, tdlm, tsl, worktrees, etc — bash-authored but zsh-compatible)
for f in "$OMARCHY_PATH"/default/bash/fns/*; do source "$f"; done

# Aliases
source ~/.config/shell/aliases.sh

# GO
export PATH="$PATH:$HOME/.go/bin"

# dotnet
export PATH="$PATH:$HOME/.dotnet"
export PATH="$PATH:$HOME/.dotnet/tools"
export DOTNET_CLI_TELEMETRY_OPTOUT=true

# pnpm
export PNPM_HOME="/home/james/.local/share/pnpm"
case ":$PATH:" in
  *":$PNPM_HOME/bin:"*) ;;
  *) export PATH="$PNPM_HOME/bin:$PATH" ;;
esac
# pnpm end

# rider
export PATH="$PATH:/home/james/rider/bin"
alias clip='xclip -selection clipboard'

# azure
export FUNCTIONS_CORE_TOOLS_TELEMETRY_OUTPUT=true

alias lg="lazygit"

# database strings
export KUMO_PROD_POSTGRES='postgresql://read_only@kumo-prod-postgresql.postgres.database.azure.com:5432/kumo_production'
source /usr/share/zsh/plugins/zsh-syntax-highlighting/zsh-syntax-highlighting.zsh
