# HTTP filesystem (FUSE)

Uses directory listings to build a directory structure and downloads the files using HTTP as you read them.

## Prerequisites

 - FUSE support on your distro and the necessary libs (should be by default on Linux).
 - Go 1.15+

## Building

```
go build
```

## Running after build

```
./go-httpfs -url "https://....." mountdir
```

## License

All source files in this project/repository are licensed under the GPLv3 license.

```
    HTTP FUSE filesystem
    Copyright (C) 2021  Simão Gomes Viana

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

