# If you come from bash you might have to change your $PATH.
# export PATH=$HOME/bin:$HOME/.local/bin:/usr/local/bin:$PATH

# Environment
export BAT_THEME=ansi
export MANROFFOPT="-c"
export MANPAGER="sh -c 'col -bx | bat -l man -p'"
export SUDO_EDITOR="$EDITOR"
export OMARCHY_PATH=$HOME/.local/share/omarchy
export PATH=$OMARCHY_PATH/bin:$PATH:$HOME/.local/bin

# Completions
autoload -Uz compinit && compinit

# Tool initialization
if command -v mise &> /dev/null; then eval "$(mise activate zsh)"; fi
if [[ ${TERM:-} != "dumb" ]] && command -v starship &> /dev/null; then eval "$(starship init zsh)"; fi
if command -v zoxide &> /dev/null; then eval "$(zoxide init zsh)"; fi
if command -v fzf &> /dev/null; then source <(fzf --zsh) 2>/dev/null; fi
if command -v gh &> /dev/null; then eval "$(gh completion -s zsh)"; fi
if command -v pnpm &> /dev/null; then eval "$(pnpm completion zsh)"; fi

# Aliases
source ~/.config/shell/aliases.sh

# GO
export PATH="$PATH:/usr/local/go/bin:$HOME/.go/bin"

# dotnet
export PATH="$PATH:$HOME/.dotnet"
export PATH="$PATH:$HOME/.dotnet/tools"
export DOTNET_CLI_TELEMETRY_OPTOUT=true

# mise shims (includes corepack and other global npm bins)
export PATH="$HOME/.local/share/mise/shims:$PATH"

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


