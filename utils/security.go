package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
)

var RANDOM_BYTES = []byte{35, 46, 57, 24, 85, 35, 24, 74, 87, 35, 88, 98, 66, 32, 14, 05}

// cipherSecret reads the key on every call rather than at package init. Package
// level vars are initialised before main() runs, which would capture an empty
// key before the .env file has been loaded.
func cipherSecret() []byte {
	return []byte(os.Getenv("ENCRYPTION_KEY"))
}

func Encrypt(text string) string {
	block, err := aes.NewCipher(cipherSecret())
	if err != nil {
		fmt.Println(err)
		return ""
	}
	plainText := []byte(text)
	cfb := cipher.NewCFBEncrypter(block, RANDOM_BYTES)
	cipherText := make([]byte, len(plainText))
	cfb.XORKeyStream(cipherText, plainText)
	return base64.StdEncoding.EncodeToString(cipherText)
}

func Decrypt(text string) string {
	block, err := aes.NewCipher(cipherSecret())
	if err != nil {
		fmt.Println(err)
		return ""
	}
	cipherText, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	cfb := cipher.NewCFBDecrypter(block, RANDOM_BYTES)
	plainText := make([]byte, len(cipherText))
	cfb.XORKeyStream(plainText, cipherText)

	return string(plainText)
}

func GenPassword() string {
	const passwordLength = 16
	const characters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@$%^&*()_+"

	buffer := make([]byte, passwordLength)

	_, err := rand.Read(buffer)
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < passwordLength; i++ {
		buffer[i] = characters[int(buffer[i])%len(characters)]
	}

	return string(buffer)
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password string, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func GenToken() string {
	const passwordLength = 16
	const characters = "abcde123456789fghijkl99232301mnopqrstuvwxyz8434212123ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	buffer := make([]byte, passwordLength)

	_, err := rand.Read(buffer)
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < passwordLength; i++ {
		buffer[i] = characters[int(buffer[i])%len(characters)]
	}

	return string(buffer)
}
