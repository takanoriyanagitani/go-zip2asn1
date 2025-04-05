#!/bin/sh

input=./sample.zip
output=./sample.output.asn1.der.dat

geninput(){
	echo generating input zip file...

	mkdir -p ./sample.d

	echo hw  > ./sample.d/hw1.txt
	echo hw2 > ./sample.d/hw2.txt
	mkdir -p ./sample.d/hw3.d

	find sample.d |
		zip \
			-r \
			-0 \
			-@ \
			-T \
			-v \
			-o \
			"${input}"
}

test -f "${input}" || geninput

export ENV_ZIP_ITEM_NAME=sample.d/hw2.txt
export ENV_ZIP_NAME="${input}"

echo converts the specified zip item to DER encoded bytes
./zipitem2asn1 |
	dd \
		if=/dev/stdin \
		of="${output}" \
		bs=1048576 \
		conv=fsync

echo
echo tries to decode the encoded zip item
cat "${output}" |
	python3 \
		-m uv \
		  run \
		  python \
		  -c 'import sys; import asn1tools; import functools; import operator; functools.reduce(
                  lambda state, f: f(state),
                  [
                      operator.methodcaller("read"),
                      functools.partial(
                          asn1tools.compile_files(
                              "./zipfile.asn",
                              "der",
                          ).decode,
                          "Item",
                      ),
                      print,
                  ],
                  sys.stdin.buffer,
		     )'
