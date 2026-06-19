# Packages

Explicitly installed packages on this system (Arch Linux / Omarchy). Run to restore:

```sh
sudo pacman -S --needed $(cat packages.md | grep '^- ' | sed 's/^- //' | tr '\n' ' ')
```

> AUR packages (installed via `yay`) are marked with `†`.

## Base System

- base
- base-devel
- linux
- linux-firmware
- linux-headers
- sudo
- man-db
- less
- unzip
- inetutils
- whois
- socat
- bc
- limine
- limine-mkinitcpio-hook
- limine-snapper-sync
- efibootmgr
- dosfstools
- exfatprogs
- btrfs-progs
- zram-generator
- kernel-modules-hook
- wireless-regdb
- dkms
- rtl8822bu-dkms
- intel-ucode
- intel-media-driver
- vpl-gpu-rt
- vulkan-intel

## Shell & Terminal

- zsh
- bash-completion
- alacritty
- tmux
- starship
- zoxide
- fzf
- eza
- bat
- fd
- ripgrep
- dust
- tldr
- gum
- usage
- plocate

## System Tools

- btop
- fastfetch
- inxi
- snapper
- ufw
- ufw-docker
- power-profiles-daemon
- bolt
- brightnessctl
- tzupdate
- expac
- fontconfig
- gnome-keyring
- polkit-gnome

## Desktop / Wayland / Hyprland

- hyprland
- hyprlock
- hypridle
- hyprsunset
- hyprpicker
- hyprland-guiutils
- hyprland-preview-share-picker
- uwsm
- waybar
- omarchy-walker
- mako
- swayosd
- swaybg
- sddm
- grim
- slurp
- satty
- gpu-screen-recorder
- xdg-desktop-portal-hyprland
- xdg-desktop-portal-gtk
- xdg-terminal-exec
- wl-clipboard
- playerctl
- pamixer
- qt5-wayland

## Fonts

- noto-fonts
- noto-fonts-cjk
- noto-fonts-emoji
- ttf-ia-writer
- ttf-jetbrains-mono-nerd
- woff2-font-awesome

## Themes & Icons

- gnome-themes-extra
- yaru-icon-theme
- kvantum-qt5

## Audio & Video

- pipewire
- pipewire-alsa
- pipewire-jack
- pipewire-pulse
- wireplumber
- gst-plugin-pipewire
- libpulse
- alsa-utils
- mpv
- obs-studio
- kdenlive
- ffmpegthumbnailer
- imagemagick
- wiremix

## Networking & Bluetooth

- iwd
- impala
- nss-mdns
- bluetui

## Input

- fcitx5
- fcitx5-gtk
- fcitx5-qt

## File Management

- nautilus
- nautilus-python
- sushi
- gvfs-mtp
- gvfs-nfs
- gvfs-smb

## Applications

- brave-origin-beta-bin †
- chromium
- obsidian †
- signal-desktop
- spotify †
- libreoffice-fresh
- evince
- gnome-calculator
- gnome-disk-utility
- pinta
- localsend
- simple-scan
- xournalpp
- imv
- cups
- cups-browsed
- cups-filters
- cups-pdf
- system-config-printer

## Development

- git
- github-cli
- neovim
- omarchy-nvim
- lazygit
- docker
- docker-buildx
- docker-compose
- lazydocker
- mise
- jq
- xmlstarlet
- stow
- python-gobject
- python-poetry-core
- python-terminaltexteffects
- luarocks
- ruby
- rust
- clang
- llvm
- dotnet-sdk
- dotnet-runtime-9.0
- mariadb-libs
- postgresql-libs
- libyaml
- libqalculate
- tesseract
- tesseract-data-eng
- tree-sitter-cli

## Omarchy

- omarchy-keyring
- aether †

## Package Management

- yay †
