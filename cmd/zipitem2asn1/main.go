package main

import (
	"context"
	"fmt"
	"log"
	"os"

	za "github.com/takanoriyanagitani/go-zip2asn1"
	. "github.com/takanoriyanagitani/go-zip2asn1/util"
)

var envValByKey func(string) IO[string] = Lift(
	func(key string) (string, error) {
		val, found := os.LookupEnv(key)
		switch found {
		case true:
			return val, nil
		default:
			return "", fmt.Errorf("env var %s missing", key)
		}
	},
)

var itemName IO[string] = envValByKey("ENV_ZIP_ITEM_NAME")

var zipName IO[string] = envValByKey("ENV_ZIP_NAME")

var rdr2zfile za.ReaderToZipFile = za.GetItemOrNil

var cfg IO[za.ZipItemConfig] = Bind(
	All(
		zipName,
		itemName,
	),
	Lift(func(s []string) (za.ZipItemConfig, error) {
		return za.ZipItemConfig{
			ZipName:         s[0],
			ItemName:        s[1],
			ReaderToZipFile: rdr2zfile,
		}, nil
	}),
)

var zitem IO[za.ZipItem] = Bind(
	cfg,
	Lift(za.GetZipItemFs),
)

var aitem IO[za.Asn1ZipFile] = Bind(
	zitem,
	Lift(func(z za.ZipItem) (za.Asn1ZipFile, error) { return z.ToAsn1(), nil }),
)

var deritem IO[[]byte] = Bind(
	aitem,
	Lift(func(a za.Asn1ZipFile) ([]byte, error) { return a.ToDer() }),
)

func bytes2stdout(dat []byte) IO[Void] {
	return func(_ context.Context) (Void, error) {
		_, e := os.Stdout.Write(dat)
		return Empty, e
	}
}

var zip2item2der2stdout IO[Void] = Bind(
	deritem,
	bytes2stdout,
)

func main() {
	_, e := zip2item2der2stdout(context.Background())
	if nil != e {
		log.Printf("%v\n", e)
	}
}
