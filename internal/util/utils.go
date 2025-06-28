package util

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"log"
	"strings"
)

func CheckPassword(password string, hashedpassword string) error {

	Hpassword, err := Crypto(password)
	if len(password) == 0 {
		return errors.New("password is emtpy")
	}
	if err != nil {

		return err
	}

	if strings.EqualFold(Hpassword, hashedpassword) {
		return nil
	} else {
		return errors.New("password is wrong")
	}

}

func Crypto(password string) (string, error) {
	sercet := "thisN0Sercet"
	salt := "HHH"
	key := sercet + salt
	origData := []byte(password)
	k := []byte(key)

	// 分组秘钥
	block, err := aes.NewCipher(k)
	if err != nil {
		log.Println("Crypto error: ", err.Error())
		return "", err
	}
	// 获取秘钥块的长度
	blockSize := block.BlockSize()
	// 补全码
	origData = PKCS7Padding(origData, blockSize)
	// 加密模式
	blockMode := cipher.NewCBCEncrypter(block, k[:blockSize])
	// 创建数组
	cryted := make([]byte, len(origData))
	// 加密
	blockMode.CryptBlocks(cryted, origData)

	return base64.StdEncoding.EncodeToString(cryted), nil

}

func PKCS7Padding(ciphertext []byte, blocksize int) []byte {
	padding := blocksize - len(ciphertext)%blocksize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}
