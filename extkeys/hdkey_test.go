package extkeys

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	masterPrivKey1 = "xprv9s21ZrQH143K3QTDL4LXw2F7HEK3wJUD2nW2nRk4stbPy6cq3jPPqjiChkVvvNKmPGJxWUtg6LnF5kejMRNNU3TGtRBeJgk33yuGBxrMPHi"
	masterPrivKey2 = "xprv9s21ZrQH143K31xYSDQpPDxsXRTUcvj2iNHm5NUtrGiGG5e2DtALGdso3pGz6ssrdK4PFmM8NSpSBHNqPqm55Qn3LqFtT2emdEXVYsCzC2U"
)

func TestBIP32Vectors(t *testing.T) {
	// Test vectors 1, 2, and 3 are taken from the BIP32 specs:
	// https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki#test-vectors
	tests := []struct {
		name    string
		seed    string
		path    []uint32
		pubKey  string
		privKey string
	}{
		// Test vector 1
		{
			"test vector 1 chain m",
			"000102030405060708090a0b0c0d0e0f",
			[]uint32{},
			"xpub661MyMwAqRbcFtXgS5sYJABqqG9YLmC4Q1Rdap9gSE8NqtwybGhePY2gZ29ESFjqJoCu1Rupje8YtGqsefD265TMg7usUDFdp6W1EGMcet8",
			masterPrivKey1,
		},
		{
			"test vector 1 chain m/0H",
			"000102030405060708090a0b0c0d0e0f",
			[]uint32{HardenedKeyStart},
			"xpub68Gmy5EdvgibQVfPdqkBBCHxA5htiqg55crXYuXoQRKfDBFA1WEjWgP6LHhwBZeNK1VTsfTFUHCdrfp1bgwQ9xv5ski8PX9rL2dZXvgGDnw",
			"xprv9uHRZZhk6KAJC1avXpDAp4MDc3sQKNxDiPvvkX8Br5ngLNv1TxvUxt4cV1rGL5hj6KCesnDYUhd7oWgT11eZG7XnxHrnYeSvkzY7d2bhkJ7",
		},
		{
			"test vector 1 chain m/0H/1",
			"000102030405060708090a0b0c0d0e0f",
			[]uint32{HardenedKeyStart, 1},
			"xpub6ASuArnXKPbfEwhqN6e3mwBcDTgzisQN1wXN9BJcM47sSikHjJf3UFHKkNAWbWMiGj7Wf5uMash7SyYq527Hqck2AxYysAA7xmALppuCkwQ",
			"xprv9wTYmMFdV23N2TdNG573QoEsfRrWKQgWeibmLntzniatZvR9BmLnvSxqu53Kw1UmYPxLgboyZQaXwTCg8MSY3H2EU4pWcQDnRnrVA1xe8fs",
		},
		{
			"test vector 1 chain m/0H/1/2H",
			"000102030405060708090a0b0c0d0e0f",
			[]uint32{HardenedKeyStart, 1, HardenedKeyStart + 2},
			"xpub6D4BDPcP2GT577Vvch3R8wDkScZWzQzMMUm3PWbmWvVJrZwQY4VUNgqFJPMM3No2dFDFGTsxxpG5uJh7n7epu4trkrX7x7DogT5Uv6fcLW5",
			"xprv9z4pot5VBttmtdRTWfWQmoH1taj2axGVzFqSb8C9xaxKymcFzXBDptWmT7FwuEzG3ryjH4ktypQSAewRiNMjANTtpgP4mLTj34bhnZX7UiM",
		},
		{
			"test vector 1 chain m/0H/1/2H/2",
			"000102030405060708090a0b0c0d0e0f",
			[]uint32{HardenedKeyStart, 1, HardenedKeyStart + 2, 2},
			"xpub6FHa3pjLCk84BayeJxFW2SP4XRrFd1JYnxeLeU8EqN3vDfZmbqBqaGJAyiLjTAwm6ZLRQUMv1ZACTj37sR62cfN7fe5JnJ7dh8zL4fiyLHV",
			"xprvA2JDeKCSNNZky6uBCviVfJSKyQ1mDYahRjijr5idH2WwLsEd4Hsb2Tyh8RfQMuPh7f7RtyzTtdrbdqqsunu5Mm3wDvUAKRHSC34sJ7in334",
		},
		{
			"test vector 1 chain m/0H/1/2H/2/1000000000",
			"000102030405060708090a0b0c0d0e0f",
			[]uint32{HardenedKeyStart, 1, HardenedKeyStart + 2, 2, 1000000000},
			"xpub6H1LXWLaKsWFhvm6RVpEL9P4KfRZSW7abD2ttkWP3SSQvnyA8FSVqNTEcYFgJS2UaFcxupHiYkro49S8yGasTvXEYBVPamhGW6cFJodrTHy",
			"xprvA41z7zogVVwxVSgdKUHDy1SKmdb533PjDz7J6N6mV6uS3ze1ai8FHa8kmHScGpWmj4WggLyQjgPie1rFSruoUihUZREPSL39UNdE3BBDu76",
		},
		// Test vector 2
		{
			"test vector 2 chain m",
			"fffcf9f6f3f0edeae7e4e1dedbd8d5d2cfccc9c6c3c0bdbab7b4b1aeaba8a5a29f9c999693908d8a8784817e7b7875726f6c696663605d5a5754514e4b484542",
			[]uint32{},
			"xpub661MyMwAqRbcFW31YEwpkMuc5THy2PSt5bDMsktWQcFF8syAmRUapSCGu8ED9W6oDMSgv6Zz8idoc4a6mr8BDzTJY47LJhkJ8UB7WEGuduB",
			masterPrivKey2,
		},
		{
			"test vector 2 chain m/0",
			"fffcf9f6f3f0edeae7e4e1dedbd8d5d2cfccc9c6c3c0bdbab7b4b1aeaba8a5a29f9c999693908d8a8784817e7b7875726f6c696663605d5a5754514e4b484542",
			[]uint32{0},
			"xpub69H7F5d8KSRgmmdJg2KhpAK8SR3DjMwAdkxj3ZuxV27CprR9LgpeyGmXUbC6wb7ERfvrnKZjXoUmmDznezpbZb7ap6r1D3tgFxHmwMkQTPH",
			"xprv9vHkqa6EV4sPZHYqZznhT2NPtPCjKuDKGY38FBWLvgaDx45zo9WQRUT3dKYnjwih2yJD9mkrocEZXo1ex8G81dwSM1fwqWpWkeS3v86pgKt",
		},
		{
			"test vector 2 chain m/0/2147483647H",
			"fffcf9f6f3f0edeae7e4e1dedbd8d5d2cfccc9c6c3c0bdbab7b4b1aeaba8a5a29f9c999693908d8a8784817e7b7875726f6c696663605d5a5754514e4b484542",
			[]uint32{0, HardenedKeyStart + 2147483647},
			"xpub6ASAVgeehLbnwdqV6UKMHVzgqAG8Gr6riv3Fxxpj8ksbH9ebxaEyBLZ85ySDhKiLDBrQSARLq1uNRts8RuJiHjaDMBU4Zn9h8LZNnBC5y4a",
			"xprv9wSp6B7kry3Vj9m1zSnLvN3xH8RdsPP1Mh7fAaR7aRLcQMKTR2vidYEeEg2mUCTAwCd6vnxVrcjfy2kRgVsFawNzmjuHc2YmYRmagcEPdU9",
		},
		{
			"test vector 2 chain m/0/2147483647H/1",
			"fffcf9f6f3f0edeae7e4e1dedbd8d5d2cfccc9c6c3c0bdbab7b4b1aeaba8a5a29f9c999693908d8a8784817e7b7875726f6c696663605d5a5754514e4b484542",
			[]uint32{0, HardenedKeyStart + 2147483647, 1},
			"xpub6DF8uhdarytz3FWdA8TvFSvvAh8dP3283MY7p2V4SeE2wyWmG5mg5EwVvmdMVCQcoNJxGoWaU9DCWh89LojfZ537wTfunKau47EL2dhHKon",
			"xprv9zFnWC6h2cLgpmSA46vutJzBcfJ8yaJGg8cX1e5StJh45BBciYTRXSd25UEPVuesF9yog62tGAQtHjXajPPdbRCHuWS6T8XA2ECKADdw4Ef",
		},
		{
			"test vector 2 chain m/0/2147483647H/1/2147483646H",
			"fffcf9f6f3f0edeae7e4e1dedbd8d5d2cfccc9c6c3c0bdbab7b4b1aeaba8a5a29f9c999693908d8a8784817e7b7875726f6c696663605d5a5754514e4b484542",
			[]uint32{0, HardenedKeyStart + 2147483647, 1, HardenedKeyStart + 2147483646},
			"xpub6ERApfZwUNrhLCkDtcHTcxd75RbzS1ed54G1LkBUHQVHQKqhMkhgbmJbZRkrgZw4koxb5JaHWkY4ALHY2grBGRjaDMzQLcgJvLJuZZvRcEL",
			"xprvA1RpRA33e1JQ7ifknakTFpgNXPmW2YvmhqLQYMmrj4xJXXWYpDPS3xz7iAxn8L39njGVyuoseXzU6rcxFLJ8HFsTjSyQbLYnMpCqE2VbFWc",
		},
		{
			"test vector 2 chain m/0/2147483647H/1/2147483646H/2",
			"fffcf9f6f3f0edeae7e4e1dedbd8d5d2cfccc9c6c3c0bdbab7b4b1aeaba8a5a29f9c999693908d8a8784817e7b7875726f6c696663605d5a5754514e4b484542",
			[]uint32{0, HardenedKeyStart + 2147483647, 1, HardenedKeyStart + 2147483646, 2},
			"xpub6FnCn6nSzZAw5Tw7cgR9bi15UV96gLZhjDstkXXxvCLsUXBGXPdSnLFbdpq8p9HmGsApME5hQTZ3emM2rnY5agb9rXpVGyy3bdW6EEgAtqt",
			"xprvA2nrNbFZABcdryreWet9Ea4LvTJcGsqrMzxHx98MMrotbir7yrKCEXw7nadnHM8Dq38EGfSh6dqA9QWTyefMLEcBYJUuekgW4BYPJcr9E7j",
		},
		// Test vector 3
		{
			"test vector 3 chain m",
			"4b381541583be4423346c643850da4b320e46a87ae3d2a4e6da11eba819cd4acba45d239319ac14f863b8d5ab5a0d0c64d2e8a1e7d1457df2e5a3c51c73235be",
			[]uint32{},
			"xpub661MyMwAqRbcEZVB4dScxMAdx6d4nFc9nvyvH3v4gJL378CSRZiYmhRoP7mBy6gSPSCYk6SzXPTf3ND1cZAceL7SfJ1Z3GC8vBgp2epUt13",
			"xprv9s21ZrQH143K25QhxbucbDDuQ4naNntJRi4KUfWT7xo4EKsHt2QJDu7KXp1A3u7Bi1j8ph3EGsZ9Xvz9dGuVrtHHs7pXeTzjuxBrCmmhgC6",
		},
		{
			"test vector 3 chain m/0H",
			"4b381541583be4423346c643850da4b320e46a87ae3d2a4e6da11eba819cd4acba45d239319ac14f863b8d5ab5a0d0c64d2e8a1e7d1457df2e5a3c51c73235be",
			[]uint32{HardenedKeyStart},
			"xpub68NZiKmJWnxxS6aaHmn81bvJeTESw724CRDs6HbuccFQN9Ku14VQrADWgqbhhTHBaohPX4CjNLf9fq9MYo6oDaPPLPxSb7gwQN3ih19Zm4Y",
			"xprv9uPDJpEQgRQfDcW7BkF7eTya6RPxXeJCqCJGHuCJ4GiRVLzkTXBAJMu2qaMWPrS7AANYqdq6vcBcBUdJCVVFceUvJFjaPdGZ2y9WACViL4L",
		},
	}

tests:
	for i, test := range tests {
		seed, err := hex.DecodeString(test.seed)
		if err != nil {
			t.Errorf("DecodeString #%d (%s): %v", i, test.name, err)
			continue
		}

		extKey, err := NewMaster(seed)
		if err != nil {
			t.Errorf("NewMasterKey #%d (%s): %v", i, test.name, err)
			continue
		}

		if !extKey.IsPrivate {
			t.Error("Master node must feature private key")
			continue
		}

		extKey, err = extKey.Derive(test.path)
		if err != nil {
			t.Errorf("cannot derive child: %v", err)
			continue tests
		}

		privKeyStr := extKey.String()
		if privKeyStr != test.privKey {
			t.Errorf("%d (%s): private key mismatch (expects: %s, got: %s)", i, test.name, test.privKey, privKeyStr)
			continue
		} else {
			t.Logf("test %d (%s): %s", i, test.name, extKey.String())
		}

		pubKey, err := extKey.Neuter()
		if err != nil {
			t.Errorf("failed to Neuter key #%d (%s): %v", i, test.name, err)
			return
		}

		// neutering twice should have no effect
		pubKey, err = pubKey.Neuter()
		if err != nil {
			t.Errorf("failed to Neuter key #%d (%s): %v", i, test.name, err)
			return
		}

		pubKeyStr := pubKey.String()
		if pubKeyStr != test.pubKey {
			t.Errorf("%d (%s): public key mismatch (expects: %s, got: %s)", i, test.name, test.pubKey, pubKeyStr)
			continue
		} else {
			t.Logf("test %d (%s, public): %s", i, test.name, extKey.String())
		}

	}
}

func TestChildDerivation(t *testing.T) {
	type testCase struct {
		name    string
		master  string
		path    []uint32
		wantKey string
	}

	// derive public keys from private keys
	getPrivateChildDerivationTests := func() []testCase {
		// The private extended keys for test vectors in [BIP32].
		testVec1MasterPrivKey := masterPrivKey1
		testVec2MasterPrivKey := masterPrivKey2

		return []testCase{
			// Test vector 1
			{
				name:    "test vector 1 chain m",
				master:  testVec1MasterPrivKey,
				path:    []uint32{},
				wantKey: masterPrivKey1,
			},
			{
				name:    "test vector 1 chain m/0",
				master:  testVec1MasterPrivKey,
				path:    []uint32{0},
				wantKey: "xprv9uHRZZhbkedL37eZEnyrNsQPFZYRAvjy5rt6M1nbEkLSo378x1CQQLo2xxBvREwiK6kqf7GRNvsNEchwibzXaV6i5GcsgyjBeRguXhKsi4R",
			},
			{
				name:    "test vector 1 chain m/0/1",
				master:  testVec1MasterPrivKey,
				path:    []uint32{0, 1},
				wantKey: "xprv9ww7sMFLzJMzy7bV1qs7nGBxgKYrgcm3HcJvGb4yvNhT9vxXC7eX7WVULzCfxucFEn2TsVvJw25hH9d4mchywguGQCZvRgsiRaTY1HCqN8G",
			},
			{
				name:    "test vector 1 chain m/0/1/2",
				master:  testVec1MasterPrivKey,
				path:    []uint32{0, 1, 2},
				wantKey: "xprv9xrdP7iD2L1YZCgR9AecDgpDMZSTzP5KCfUykGXgjBxLgp1VFHsEeL3conzGAkbc1MigG1o8YqmfEA2jtkPdf4vwMaGJC2YSDbBTPAjfRUi",
			},
			{
				name:    "test vector 1 chain m/0/1/2/2",
				master:  testVec1MasterPrivKey,
				path:    []uint32{0, 1, 2, 2},
				wantKey: "xprvA2J8Hq4eiP7xCEBP7gzRJGJnd9CHTkEU6eTNMrZ6YR7H5boik8daFtDZxmJDfdMSKHwroCfAfsBKWWidRfBQjpegy6kzXSkQGGoMdWKz5Xh",
			},
			{
				name:    "test vector 1 chain m/0/1/2/2/1000000000",
				master:  testVec1MasterPrivKey,
				path:    []uint32{0, 1, 2, 2, 1000000000},
				wantKey: "xprvA3XhazxncJqJsQcG85Gg61qwPQKiobAnWjuPpjKhExprZjfse6nErRwTMwGe6uGWXPSykZSTiYb2TXAm7Qhwj8KgRd2XaD21Styu6h6AwFz",
			},

			// Test vector 2
			{
				name:    "test vector 2 chain m",
				master:  testVec2MasterPrivKey,
				path:    []uint32{},
				wantKey: masterPrivKey2,
			},
			{
				name:    "test vector 2 chain m/0",
				master:  testVec2MasterPrivKey,
				path:    []uint32{0},
				wantKey: "xprv9vHkqa6EV4sPZHYqZznhT2NPtPCjKuDKGY38FBWLvgaDx45zo9WQRUT3dKYnjwih2yJD9mkrocEZXo1ex8G81dwSM1fwqWpWkeS3v86pgKt",
			},
			{
				name:    "test vector 2 chain m/0/2147483647",
				master:  testVec2MasterPrivKey,
				path:    []uint32{0, 2147483647},
				wantKey: "xprv9wSp6B7cXJWXZRpDbxkFg3ry2fuSyUfvboJ5Yi6YNw7i1bXmq9QwQ7EwMpeG4cK2pnMqEx1cLYD7cSGSCtruGSXC6ZSVDHugMsZgbuY62m6",
			},
			{
				name:    "test vector 2 chain m/0/2147483647/1",
				master:  testVec2MasterPrivKey,
				path:    []uint32{0, 2147483647, 1},
				wantKey: "xprv9ysS5br6UbWCRCJcggvpUNMyhVWgD7NypY9gsVTMYmuRtZg8izyYC5Ey4T931WgWbfJwRDwfVFqV3b29gqHDbuEpGcbzf16pdomk54NXkSm",
			},
			{
				name:    "test vector 2 chain m/0/2147483647/1/2147483646",
				master:  testVec2MasterPrivKey,
				path:    []uint32{0, 2147483647, 1, 2147483646},
				wantKey: "xprvA2LfeWWwRCxh4iqigcDMnUf2E3nVUFkntc93nmUYBtb9rpSPYWa8MY3x9ZHSLZkg4G84UefrDruVK3FhMLSJsGtBx883iddHNuH1LNpRrEp",
			},
			{
				name:    "test vector 2 chain m/0/2147483647/1/2147483646/2",
				master:  testVec2MasterPrivKey,
				path:    []uint32{0, 2147483647, 1, 2147483646, 2},
				wantKey: "xprvA48ALo8BDjcRET68R5RsPzF3H7WeyYYtHcyUeLRGBPHXu6CJSGjwW7dWoeUWTEzT7LG3qk6Eg6x2ZoqD8gtyEFZecpAyvchksfLyg3Zbqam",
			},

			// Custom tests to trigger specific conditions.
			{
				// Seed 000000000000000000000000000000da.
				name:    "Derived privkey with zero high byte m/0",
				master:  "xprv9s21ZrQH143K4FR6rNeqEK4EBhRgLjWLWhA3pw8iqgAKk82ypz58PXbrzU19opYcxw8JDJQF4id55PwTsN1Zv8Xt6SKvbr2KNU5y8jN8djz",
				path:    []uint32{0},
				wantKey: "xprv9uC5JqtViMmgcAMUxcsBCBFA7oYCNs4bozPbyvLfddjHou4rMiGEHipz94xNaPb1e4f18TRoPXfiXx4C3cDAcADqxCSRSSWLvMBRWPctSN9",
			},
		}

	}

	// derive public keys from other public keys
	getPublicChildDerivationTests := func() []testCase {
		// The public extended keys for test vectors in [BIP32].
		testVec1MasterPubKey := "xpub661MyMwAqRbcFtXgS5sYJABqqG9YLmC4Q1Rdap9gSE8NqtwybGhePY2gZ29ESFjqJoCu1Rupje8YtGqsefD265TMg7usUDFdp6W1EGMcet8"
		testVec2MasterPubKey := "xpub661MyMwAqRbcFW31YEwpkMuc5THy2PSt5bDMsktWQcFF8syAmRUapSCGu8ED9W6oDMSgv6Zz8idoc4a6mr8BDzTJY47LJhkJ8UB7WEGuduB"

		return []testCase{
			// Test vector 1
			{
				name:    "test vector 1 chain m",
				master:  testVec1MasterPubKey,
				path:    []uint32{},
				wantKey: "xpub661MyMwAqRbcFtXgS5sYJABqqG9YLmC4Q1Rdap9gSE8NqtwybGhePY2gZ29ESFjqJoCu1Rupje8YtGqsefD265TMg7usUDFdp6W1EGMcet8",
			},
			{
				name:    "test vector 1 chain m/0",
				master:  testVec1MasterPubKey,
				path:    []uint32{0},
				wantKey: "xpub68Gmy5EVb2BdFbj2LpWrk1M7obNuaPTpT5oh9QCCo5sRfqSHVYWex97WpDZzszdzHzxXDAzPLVSwybe4uPYkSk4G3gnrPqqkV9RyNzAcNJ1",
			},
			{
				name:    "test vector 1 chain m/0/1",
				master:  testVec1MasterPubKey,
				path:    []uint32{0, 1},
				wantKey: "xpub6AvUGrnEpfvJBbfx7sQ89Q8hEMPM65UteqEX4yUbUiES2jHfjexmfJoxCGSwFMZiPBaKQT1RiKWrKfuDV4vpgVs4Xn8PpPTR2i79rwHd4Zr",
			},
			{
				name:    "test vector 1 chain m/0/1/2",
				master:  testVec1MasterPubKey,
				path:    []uint32{0, 1, 2},
				wantKey: "xpub6BqyndF6rhZqmgktFCBcapkwubGxPqoAZtQaYewJHXVKZcLdnqBVC8N6f6FSHWUghjuTLeubWyQWfJdk2G3tGgvgj3qngo4vLTnnSjAZckv",
			},
			{
				name:    "test vector 1 chain m/0/1/2/2",
				master:  testVec1MasterPubKey,
				path:    []uint32{0, 1, 2, 2},
				wantKey: "xpub6FHUhLbYYkgFQiFrDiXRfQFXBB2msCxKTsNyAExi6keFxQ8sHfwpogY3p3s1ePSpUqLNYks5T6a3JqpCGszt4kxbyq7tUoFP5c8KWyiDtPp",
			},
			{
				name:    "test vector 1 chain m/0/1/2/2/1000000000",
				master:  testVec1MasterPubKey,
				path:    []uint32{0, 1, 2, 2, 1000000000},
				wantKey: "xpub6GX3zWVgSgPc5tgjE6ogT9nfwSADD3tdsxpzd7jJoJMqSY12Be6VQEFwDCp6wAQoZsH2iq5nNocHEaVDxBcobPrkZCjYW3QUmoDYzMFBDu9",
			},

			// Test vector 2
			{
				name:    "test vector 2 chain m",
				master:  testVec2MasterPubKey,
				path:    []uint32{},
				wantKey: "xpub661MyMwAqRbcFW31YEwpkMuc5THy2PSt5bDMsktWQcFF8syAmRUapSCGu8ED9W6oDMSgv6Zz8idoc4a6mr8BDzTJY47LJhkJ8UB7WEGuduB",
			},
			{
				name:    "test vector 2 chain m/0",
				master:  testVec2MasterPubKey,
				path:    []uint32{0},
				wantKey: "xpub69H7F5d8KSRgmmdJg2KhpAK8SR3DjMwAdkxj3ZuxV27CprR9LgpeyGmXUbC6wb7ERfvrnKZjXoUmmDznezpbZb7ap6r1D3tgFxHmwMkQTPH",
			},
			{
				name:    "test vector 2 chain m/0/2147483647",
				master:  testVec2MasterPubKey,
				path:    []uint32{0, 2147483647},
				wantKey: "xpub6ASAVgeWMg4pmutghzHG3BohahjwNwPmy2DgM6W9wGegtPrvNgjBwuZRD7hSDFhYfunq8vDgwG4ah1gVzZysgp3UsKz7VNjCnSUJJ5T4fdD",
			},
			{
				name:    "test vector 2 chain m/0/2147483647/1",
				master:  testVec2MasterPubKey,
				path:    []uint32{0, 2147483647, 1},
				wantKey: "xpub6CrnV7NzJy4VdgP5niTpqWJiFXMAca6qBm5Hfsry77SQmN1HGYHnjsZSujoHzdxf7ZNK5UVrmDXFPiEW2ecwHGWMFGUxPC9ARipss9rXd4b",
			},
			{
				name:    "test vector 2 chain m/0/2147483647/1/2147483646",
				master:  testVec2MasterPubKey,
				path:    []uint32{0, 2147483647, 1, 2147483646},
				wantKey: "xpub6FL2423qFaWzHCvBndkN9cbkn5cysiUeFq4eb9t9kE88jcmY63tNuLNRzpHPdAM4dUpLhZ7aUm2cJ5zF7KYonf4jAPfRqTMTRBNkQL3Tfta",
			},
			{
				name:    "test vector 2 chain m/0/2147483647/1/2147483646/2",
				master:  testVec2MasterPubKey,
				path:    []uint32{0, 2147483647, 1, 2147483646, 2},
				wantKey: "xpub6H7WkJf547AiSwAbX6xsm8Bmq9M9P1Gjequ5SipsjipWmtXSyp4C3uwzewedGEgAMsDy4jEvNTWtxLyqqHY9C12gaBmgUdk2CGmwachwnWK",
			},
		}
	}

	runTests := func(tests []testCase) {
		for i, test := range tests {
			extKey, err := NewKeyFromString(test.master)
			if err != nil {
				t.Errorf("NewKeyFromString #%d (%s): unexpected error creating extended key: %v", i, test.name, err)
				continue
			}
			extKey, err = extKey.Derive(test.path)
			if err != nil {
				t.Errorf("cannot derive child: %v", err)
				continue
			}

			gotKey := extKey.String()
			if gotKey != test.wantKey {
				t.Errorf("Child #%d (%s): mismatched serialized extended key -- got: %s, want: %s", i, test.name, gotKey, test.wantKey)
				continue
			} else {
				t.Logf("test %d (%s): %s", i, test.name, extKey.String())
			}
		}
	}

	runTests(getPrivateChildDerivationTests())
	runTests(getPublicChildDerivationTests())
}

func TestErrors(t *testing.T) {
	// Should get an error when seed has too few bytes.
	_, err := NewMaster(bytes.Repeat([]byte{0x00}, 15))
	if err != ErrInvalidSeedLen {
		t.Errorf("NewMaster: mismatched error -- got: %v, want: %v",
			err, ErrInvalidSeedLen)
	}

	// Should get an error when seed has too many bytes.
	_, err = NewMaster(bytes.Repeat([]byte{0x00}, 65))
	if err != ErrInvalidSeedLen {
		t.Errorf("NewMaster: mismatched error -- got: %v, want: %v",
			err, ErrInvalidSeedLen)
	}

	// Generate a new key and neuter it to a public extended key.
	mnemonic := NewMnemonic()

	phrase, err := mnemonic.MnemonicPhrase(128, EnglishLanguage)
	if err != nil {
		t.Errorf("Test failed: could not create seed: %s", err)
	}

	password := "badpassword"
	extKey, err := NewMaster(mnemonic.MnemonicSeed(phrase, password))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	pubKey, err := extKey.Neuter()
	if err != nil {
		t.Errorf("Neuter: unexpected error: %v", err)
		return
	}

	// Deriving a hardened child extended key should fail from a public key.
	_, err = pubKey.Child(HardenedKeyStart)
	if err != ErrDerivingHardenedFromPublic {
		t.Errorf("Child: mismatched error -- got: %v, want: %v", err, ErrDerivingHardenedFromPublic)
	}

	_, err = pubKey.BIP44Child(CoinTypeETH, 0)
	if err != ErrInvalidMasterKey {
		t.Errorf("BIP44Child: mistmatched error -- got: %v, want: %v", err, ErrInvalidMasterKey)
	}

	childKey, _ := extKey.Child(HardenedKeyStart + 1)
	_, err = childKey.BIP44Child(CoinTypeETH, 0) // this should be called from master only
	if err != ErrInvalidMasterKey {
		t.Errorf("BIP44Child: mistmatched error -- got: %v, want: %v", err, ErrInvalidMasterKey)
	}

	// NewKeyFromString failure tests.
	tests := []struct {
		name      string
		key       string
		err       error
		neuter    bool
		neuterErr error
		extKey    *ExtendedKey
	}{
		{
			name: "invalid key length",
			key:  "xpub1234",
			err:  ErrInvalidKeyLen,
		},
		{
			name: "bad checksum",
			key:  "xpub661MyMwAqRbcFtXgS5sYJABqqG9YLmC4Q1Rdap9gSE8NqtwybGhePY2gZ29ESFjqJoCu1Rupje8YtGqsefD265TMg7usUDFdp6W1EBygr15",
			err:  ErrBadChecksum,
		},
		{
			name: "pubkey not on curve",
			key:  "xpub661MyMwAqRbcFtXgS5sYJABqqG9YLmC4Q1Rdap9gSE8NqtwybGhePY2gZ1hr9Rwbk95YadvBkQXxzHBSngB8ndpW6QH7zhhsXZ2jHyZqPjk",
			err:  errors.New("pubkey isn't on secp256k1 curve"),
		},
		{
			name:      "unsupported version",
			key:       "xbad4LfUL9eKmA66w2GJdVMqhvDmYGJpTGjWRAtjHqoUY17sGaymoMV9Cm3ocn9Ud6Hh2vLFVC7KSKCRVVrqc6dsEdsTjRV1WUmkK85YEUujAPX",
			err:       nil,
			neuter:    true,
			neuterErr: chaincfg.ErrUnknownHDKeyID,
		},
		{
			name:      "zeroed extended key",
			key:       EmptyExtendedKeyString,
			err:       nil,
			neuter:    false,
			neuterErr: nil,
			extKey:    &ExtendedKey{},
		},
		{
			name:      "empty string",
			key:       "",
			err:       nil,
			neuter:    false,
			neuterErr: nil,
			extKey:    &ExtendedKey{},
		},
	}

	for i, test := range tests {
		extKey, err := NewKeyFromString(test.key)
		if !reflect.DeepEqual(err, test.err) {
			t.Errorf("NewKeyFromString #%d (%s): mismatched error -- got: %v, want: %v", i, test.name, err, test.err)
			continue
		}

		if test.neuter {
			_, err := extKey.Neuter()
			if !reflect.DeepEqual(err, test.neuterErr) {
				t.Errorf("Neuter #%d (%s): mismatched error -- got: %v, want: %v", i, test.name, err, test.neuterErr)
				continue
			}
		}

		if test.extKey != nil {
			if !reflect.DeepEqual(extKey, test.extKey) {
				t.Errorf("ExtKey #%d (%s): mismatched extended key -- got: %+v, want: %+v", i, test.name, extKey, test.extKey)
				continue
			}
		}
	}
}

func TestMaxDepth(t *testing.T) {
	mnemonic := NewMnemonic()
	phrase, err := mnemonic.MnemonicPhrase(128, EnglishLanguage)
	if err != nil {
		t.Errorf("Test failed: could not create mnemonic phrase: %v", err)
	}

	lastParentKey, err := NewMaster(mnemonic.MnemonicSeed(phrase, "test-password"))
	if err != nil {
		t.Errorf("couldn't create master extended key: %v", err)
	}

	lastParentKey.Depth = 255

	_, err = lastParentKey.Child(0)
	if err != ErrMaxDepthExceeded {
		t.Errorf("Expected ErrMaxDepthExceeded, got %+v", err)
	}
}

func TestBIP44ChildDerivation(t *testing.T) {
	keyString := masterPrivKey1
	derivedKey1String := "xprvA38t8tFW4vbuB7WJXEqMFmZqRrcZUKWqqMcGjjKjr2hbfvPhRtLLJGL4ayWG8shF1VkuUikVGodGshLiKRS7WrdsrGSVDQCY33qoPBxG2Kp"
	derivedKey2String := "xprvA38t8tFW4vbuDgBNpekPnuMSfpWziDLdF7W9Zd3mPy6eDEkM5F17vk59RtVoFbNdBBq84EJf5CqdZhhEoBkAM4DXHQsDqvUxVnncfnDQEFg"

	extKey, err := NewKeyFromString(keyString)
	if err != nil {
		t.Error("NewKeyFromString: cannot create extended key")
	}

	accounKey1, err := extKey.BIP44Child(CoinTypeETH, 0)
	if err != nil {
		t.Error("Error dering BIP44-compliant key")
	}
	if accounKey1.String() != derivedKey1String {
		t.Errorf("BIP44Child: key mismatch -- got: %v, want: %v", accounKey1.String(), derivedKey1String)
	}
	t.Logf("Account 1 key: %s", accounKey1.String())

	accounKey2, err := extKey.BIP44Child(CoinTypeETH, 1)
	if err != nil {
		t.Error("Error dering BIP44-compliant key")
	}
	if accounKey2.String() != derivedKey2String {
		t.Errorf("BIP44Child: key mismatch -- got: %v, want: %v", accounKey2.String(), derivedKey2String)
	}
	t.Logf("Account 1 key: %s", accounKey2.String())
}

func TestChildForPurpose(t *testing.T) {
	masterKey, err := NewKeyFromString(masterPrivKey1)
	if err != nil {
		t.Error("NewKeyFromString: cannot create master extended key")
	}

	bip44Child, err := masterKey.EthBIP44Child(0)
	if err != nil {
		t.Error("Error deriving BIP44-compliant key")
	}

	eip1581Child, err := masterKey.EthEIP1581ChatChild(0)
	if err != nil {
		t.Error("Error deriving EIP1581-compliant key")
	}

	walletChild, err := masterKey.ChildForPurpose(KeyPurposeWallet, 0)
	if err != nil {
		t.Error("Error dering BIP44-compliant key")
	}

	chatChild, err := masterKey.ChildForPurpose(KeyPurposeChat, 0)
	if err != nil {
		t.Error("Error dering EIP1581-compliant key")
	}

	// Check that ChildForPurpose with KeyPurposeWallet generates a BIP44 key
	if walletChild.String() != bip44Child.String() {
		t.Errorf("wrong wallet key. expected to be equal to bip44Child")
	}

	// Check that ChildForPurpose with KeyPurposeChat generates a EIP1581 key
	if chatChild.String() != eip1581Child.String() {
		t.Errorf("wrong chat key. expected to be equal to eip1581Child")
	}

	// Check that the key generated by ChildForPurpose with KeyPurposeChat is different from the BIP44
	if walletChild.String() == chatChild.String() {
		t.Errorf("wrong chat key. expected to be diferrent from the wallet key")
	}
}

func TestHDWalletCompatibility(t *testing.T) {
	password := "TREZOR"
	mnemonic := NewMnemonic()
	mnemonicPhrase := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	seed := mnemonic.MnemonicSeed(mnemonicPhrase, password)
	rootKey, err := NewMaster(seed)
	if err != nil {
		t.Errorf("couldn't create master extended key: %v", err)
	}

	expectedAddresses := []struct {
		address string
		pubKey  string
		privKey string
	}{
		{
			address: "0x9c32F71D4DB8Fb9e1A58B0a80dF79935e7256FA6",
			pubKey:  "0x03986dee3b8afe24cb8ccb2ac23dac3f8c43d22850d14b809b26d6b8aa5a1f4778",
			privKey: "0x62f1d86b246c81bdd8f6c166d56896a4a5e1eddbcaebe06480e5c0bc74c28224",
		},
		{
			address: "0x7AF7283bd1462C3b957e8FAc28Dc19cBbF2FAdfe",
			pubKey:  "0x03462e7b95dab24fe8a57ac897d9026545ec4327c9c5e4a772e5d14cc5422f9489",
			privKey: "0x49ee230b1605382ac1c40079191bca937fc30e8c2fa845b7de27a96ffcc4ddbf",
		},
		{
			address: "0x05f48E30fCb69ADcd2A591Ebc7123be8BE72D7a1",
			pubKey:  "0x036650e4b2b8e731a0ef12cda892b70cb95e78ea6e576ba995019b5e9aa7d9c0f5",
			privKey: "0xeef2c0702151930b84cffcaa642af58e692956314519114e78f3211a6465f28b",
		},
		{
			address: "0xbfE91Bc05cE66013660D7Eb742F74BD324DA5F92",
			pubKey:  "0x0201d1c12e8fcea03a68ad5fd0d02fd0a4bfe0339618f949e2e30cf311e8b83c46",
			privKey: "0xbca51d1d3529a0e0787933a2293cf46d9b973ea3ea00e28d3bd33590bc7f7156",
		},
	}

	for i := 0; i < len(expectedAddresses); i++ {
		key, err := rootKey.BIP44Child(CoinTypeETH, uint32(i))
		if err != nil {
			t.Errorf("Error deriving BIP44-compliant key: %s", err)
		}

		privateKeyECDSA := key.ToECDSA()
		address := crypto.PubkeyToAddress(privateKeyECDSA.PublicKey).Hex()

		if address != expectedAddresses[i].address {
			t.Errorf("wrong address generated. expected %s, got %s", expectedAddresses[i].address, address)
		}

		pubKey := fmt.Sprintf("0x%x", (crypto.CompressPubkey(&privateKeyECDSA.PublicKey)))
		if pubKey != expectedAddresses[i].pubKey {
			t.Errorf("wrong public key generated. expected %s, got %s", expectedAddresses[i].pubKey, pubKey)
		}

		privKey := fmt.Sprintf("0x%x", crypto.FromECDSA(privateKeyECDSA))
		if privKey != expectedAddresses[i].privKey {
			t.Errorf("wrong private key generated. expected %s, got %s", expectedAddresses[i].privKey, privKey)
		}
	}
}

// TestPrivateKeyDataWithLeadingZeros is a regression test that checks
// we don't re-introduce a bug we had in the past.
// For a specific mnemonic phrase, we were deriving a wrong key/address
// at path m/44'/60'/0'/0/0 compared to other wallets.
// In this specific case, the second child key is represented in 31 bytes.
// The problem raises when deriving its child key.
// One of the step to derive the child key is calling our splitHMAC
// that returns a secretKey and a chainCode.
// Inside this function we make a sha512 of a seed that is a 37 bytes with:
// 1 byte with 0x00
// 32 bytes for the key data
// 4 bytes for the child key index
// In our case, if the key was less then 32 bytes, it was shifted to the left of that 32 bytes space,
// resulting in a different seed, and a different data returned from the sha512 call.
// https://medium.com/@alexberegszaszi/why-do-my-bip32-wallets-disagree-6f3254cc5846#.86inuifuq
// https://github.com/iancoleman/bip39/issues/58
func TestPrivateKeyDataWithLeadingZeros(t *testing.T) {
	mn := NewMnemonic()
	words := "radar blur cabbage chef fix engine embark joy scheme fiction master release"
	key, _ := NewMaster(mn.MnemonicSeed(words, ""))

	path := []uint32{
		HardenedKeyStart + 44, // purpose
		HardenedKeyStart + 60, // cointype
		HardenedKeyStart + 0,  // account
		0,                     // change
		0,                     // index
	}

	for _, part := range path {
		key, _ = key.Child(part)
		if length := len(key.KeyData); length != 32 {
			t.Errorf("expected key length to be 32, got: %d", length)
		}
	}

	expectedAddress := "0xaC39b311DCEb2A4b2f5d8461c1cdaF756F4F7Ae9"
	address := crypto.PubkeyToAddress(key.ToECDSA().PublicKey).Hex()

	if address != expectedAddress {
		t.Errorf("expected address %s, got: %s", expectedAddress, address)
	}
}

//func TestNewKey(t *testing.T) {
//	mnemonic := NewMnemonic()
//
//	phrase, err := mnemonic.MnemonicPhrase(128, EnglishLanguage)
//	if err != nil {
//		t.Errorf("Test failed: could not create seed: %s", err)
//	}
//
//	password := "badpassword"
//	mnemonic.salt = "Bitcoin seed"
//	key, err := NewMaster(mnemonic.MnemonicSeed(phrase, password))
//	if err != nil {
//		t.Error(err)
//	}
//	t.Logf("%x", key.KeyData)
//}
