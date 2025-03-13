# PDF Info

A simple Go library focused on extracting metadata from PDF files.

## Installation

```sh
go get github.com/mahesarohman98/pdfinfo

```
## Usage

```go
package main

import (
	"fmt"
	"github.com/mahesarohman98/pdfinfo"
)

func main() {
	info, err := pdfinfo.Extract("sample.pdf")
	if err != nil {
		fmt.Println("Error extracting metadata:", err)
		return
	}

	fmt.Println("Title:", info.Key("Title").Text())
	fmt.Println("Author:", info.Key("Author").Text())
}

```

