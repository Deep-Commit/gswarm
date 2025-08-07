package main

import (
	"fmt"

	"golang.org/x/crypto/sha3"
)

func main() {
	// HFUploadSubmitted event signature
	signature := "HFUploadSubmitted(address,string,string,uint256,bool)"

	// Create Keccak256 hash
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write([]byte(signature))
	hash := hasher.Sum(nil)

	fmt.Printf("HFUploadSubmitted signature: 0x%x\n", hash)
}
