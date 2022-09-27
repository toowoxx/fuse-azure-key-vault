# Azure Key Vault FUSE File System
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Ftoowoxx%2Ffuse-azure-key-vault.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Ftoowoxx%2Ffuse-azure-key-vault?ref=badge_shield)


Displays your azure key vault in a directory structure.

## Prerequisites

 - FUSE support on your distro and the necessary libs (should be by default on Linux).
 - Go 1.19+

## Building

```
go build -o fuse.azkv
```

## Running after build

```
./fuse.azkv -url "https://....vault.azure.net" mountdir
```

## License

All source files in this project/repository are licensed under the GPLv3 license.

```
    FUSE filesystem for Az Key Vault
    Copyright (C) 2021  Simão Gomes Viana
    Copyright (C) 2022  Toowoxx IT GmbH

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with this program.  If not, see <https://www.gnu.org/licenses/>.
```



[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Ftoowoxx%2Ffuse-azure-key-vault.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Ftoowoxx%2Ffuse-azure-key-vault?ref=badge_large)