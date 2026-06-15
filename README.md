# Dotfiles

Personal dotfiles managed with [GNU Stow](https://www.gnu.org/software/stow/).

## Requirements

- [GNU Stow](https://www.gnu.org/software/stow/)

Install on Debian/Ubuntu:

```sh
sudo apt install stow
```

Install on macOS:

```sh
brew install stow
```

Install on Arch Linux:

```sh
sudo pacman -S stow
```

## Installation

Clone the repository:

```sh
# You must use the `--recurse-submodules` flag otherwise configs won't be filled
# `-j8` is an optional performance optimization
git clone --recurse-submodules -j8 https://github.com/hamst/dotfiles.git ~/dotfiles
cd ~/dotfiles

# If you didn't use the `--recurse-submodules` flag initially, intiate submodules
git submodule update --init --recursive
```

Stow the desired packages:

```sh
stow package
## or just the whole directory at once
stow .
```

## System configs (requires root)

Some configs (kernel module params, udev rules) live in `/etc/` and aren't managed by Stow.
Run the script to copy them:

```sh
./install-system.sh
```
