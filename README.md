# go-file-searcher

A simple fuzzy file searcher written in Go.

## Features

- Fuzzy search for a pattern in files under a directory.
- Supports filtering files by extension.
- Concurrent search using multiple workers.

## Usage

```sh
go run main.go -path <directory> -pattern <search-pattern> [-ext <.ext1,.ext2>] [-workers <num>]