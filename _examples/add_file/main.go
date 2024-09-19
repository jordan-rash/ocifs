package main

import "disorder.dev/ocifs"

func main() {
	rfs, err := ocifs.NewRootFS("docker.io/synadia/nex-rootfs:alpine")
	if err != nil {
		panic(err)
	}

	err = rfs.Build()
	if err != nil {
		panic(err)
	}

	err = rfs.AddFile("./hello.txt", "/home/root/hello.txt")
	if err != nil {
		panic(err)
	}

	err = rfs.Create()
	if err != nil {
		panic(err)
	}
}
