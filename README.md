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
git clone https://github.com/hamst/dotfiles.git ~/dotfiles
cd ~/dotfiles
```

Stow the desired packages:

```sh
stow package
## or just the whole directory at once
stow .
```
