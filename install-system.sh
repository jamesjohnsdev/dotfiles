#!/bin/sh
set -eu

DOTFILES="${DOTFILES:-$HOME/dotfiles}"

echo "Installing system-level configs from dotfiles (requires root)..."

# Kernel module parameters (loaded early boot via kmod)
if [ -d "$DOTFILES/.config/modprobe.d" ]; then
    sudo cp -v "$DOTFILES"/.config/modprobe.d/*.conf /etc/modprobe.d/
fi

# udev rules (hotplug events)
if [ -d "$DOTFILES/.config/udev/rules.d" ]; then
    sudo cp -v "$DOTFILES"/.config/udev/rules.d/*.rules /etc/udev/rules.d/
    sudo udevadm control --reload-rules
fi

# Brave browser policies (applies to both Brave and Brave Origin Beta)
if [ -d "$DOTFILES/.config/brave/policies/managed" ]; then
    sudo mkdir -p /etc/brave/policies/managed
    sudo cp -v "$DOTFILES"/.config/brave/policies/managed/*.json /etc/brave/policies/managed/
fi

echo "Done. Reboot or reload modules for changes to take effect."
