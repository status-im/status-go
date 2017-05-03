package extkeys_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"reflect"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/status-im/status-go/extkeys"
)

func TestBIP32Vectors(t *testing.T) {
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
			"xprv9s21ZrQH143K3QTDL4LXw2F7HEK3wJUD2nW2nRk4stbPy6cq3jPPqjiChkVvvNKmPGJxWUtg6LnF5kejMRNNU3TGtRBeJgk33yuGBxrMPHi",
		},
		{
			"test vector 1 chain m/0H",
			"000102030405060708090a0b0c0d0e0f",
			[]uint32{extkeys.HardenedKeyStart},
			"xpub68Gmy5EdvgibQVfPdqkBBCHxA5htiqg55crXYuXoQRKfDBFA1WEjWgP6LHhwBZeNK1VTsfTFUHCdrfp1bgwQ9xv5ski8PX9rL2dZXvgGDnw",
			"xprv9uHRZZhk6KAJC1avXpDAp4MDc3sQKNxDiPvvkX8Br5ngLNv1TxvUxt4cV1rGL5hj6KCesnDYUhd7oWgT11eZG7XnxHrnYeSvkzY7d2bhkJ7",
		},
		{
			"test vector 1 chain m/0H/1",
			"000102030405060708090a0b0c0d0e0f",
			[]uint32{extkeys.HardenedKeyStart, 1},
			"xpub6ASuArnXKPbfEwhqN6e3mwBcDTgzisQN1wXN9BJcM47sSikHjJf3UFHKkNAWbWMiGj7Wf5uMash7SyYq527Hqck2AxYysAA7xmALppuCkwQ",
			"xprv9wTYmMFdV23N2TdNG573QoEsfRrWKQgWeibmLntzniatZvR9BmLnvSxqu53Kw1UmYPxLgboyZQaXwTCg8MSY3H2EU4pWcQDnRnrVA1xe8fs",
		},
		{
			"test vector 1 chain m/0H/1/2H",
			"000102030405060708090a0b0c0d0e0f",
			[]uint32{extkeys.HardenedKeyStart, 1, extkeys.HardenedKeyStart + 2},
			"xpub6D4BDPcP2GT577Vvch3R8wDkScZWzQzMMUm3PWbmWvVJrZwQY4VUNgqFJPMM3No2dFDFGTsxxpG5uJh7n7epu4trkrX7x7DogT5Uv6fcLW5",
			"xprv9z4pot5VBttmtdRTWfWQmoH1taj2axGVzFqSb8C9xaxKymcFzXBDptWmT7FwuEzG3ryjH4ktypQSAewRiNMjANTtpgP4mLTj34bhnZX7UiM",
		},
		{
			"test vector 1 chain m/0H/1/2H/2",
			"000102030405060708090a0b0c0d0e0f",
			[]uint32{extkeys.HardenedKeyStart, 1, extkeys.HardenedKeyStart + 2, 2},
			"xpub6FHa3pjLCk84BayeJxFW2SP4XRrFd1JYnxeLeU8EqN3vDfZmbqBqaGJAyiLjTAwm6ZLRQUMv1ZACTj37sR62cfN7fe5JnJ7dh8zL4fiyLHV",
			"xprvA2JDeKCSNNZky6uBCviVfJSKyQ1mDYahRjijr5idH2WwLsEd4Hsb2Tyh8RfQMuPh7f7RtyzTtdrbdqqsunu5Mm3wDvUAKRHSC34sJ7in334",
		},
		{
			"test vector 1 chain m/0H/1/2H/2/1000000000",
			"000102030405060708090a0b0c0d0e0f",
			[]uint32{extkeys.HardenedKeyStart, 1, extkeys.HardenedKeyStart + 2, 2, 1000000000},
			"xpub6H1LXWLaKsWFhvm6RVpEL9P4KfRZSW7abD2ttkWP3SSQvnyA8FSVqNTEcYFgJS2UaFcxupHiYkro49S8yGasTvXEYBVPamhGW6cFJodrTHy",
			"xprvA41z7zogVVwxVSgdKUHDy1SKmdb533PjDz7J6N6mV6uS3ze1ai8FHa8kmHScGpWmj4WggLyQjgPie1rFSruoUihUZREPSL39UNdE3BBDu76",
		},
		// Test vector 2
		{
			"test vector 2 chain m",
			"fffcf9f6f3f0edeae7e4e1dedbd8d5d2cfccc9c6c3c0bdbab7b4b1aeaba8a5a29f9c999693908d8a8784817e7b7875726f6c696663605d5a5754514e4b484542",
			[]uint32{},
			"xpub661MyMwAqRbcFW31YEwpkMuc5THy2PSt5bDMsktWQcFF8syAmRUapSCGu8ED9W6oDMSgv6Zz8idoc4a6mr8BDzTJY47LJhkJ8UB7WEGuduB",
			"xprv9s21ZrQH143K31xYSDQpPDxsXRTUcvj2iNHm5NUtrGiGG5e2DtALGdso3pGz6ssrdK4PFmM8NSpSBHNqPqm55Qn3LqFtT2emdEXVYsCzC2U",
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
			[]uint32{0, extkeys.HardenedKeyStart + 2147483647},
			"xpub6ASAVgeehLbnwdqV6UKMHVzgqAG8Gr6riv3Fxxpj8ksbH9ebxaEyBLZ85ySDhKiLDBrQSARLq1uNRts8RuJiHjaDMBU4Zn9h8LZNnBC5y4a",
			"xprv9wSp6B7kry3Vj9m1zSnLvN3xH8RdsPP1Mh7fAaR7aRLcQMKTR2vidYEeEg2mUCTAwCd6vnxVrcjfy2kRgVsFawNzmjuHc2YmYRmagcEPdU9",
		},
		{
			"test vector 2 chain m/0/2147483647H/1",
			"fffcf9f6f3f0edeae7e4e1dedbd8d5d2cfccc9c6c3c0bdbab7b4b1aeaba8a5a29f9c999693908d8a8784817e7b7875726f6c696663605d5a5754514e4b484542",
			[]uint32{0, extkeys.HardenedKeyStart + 2147483647, 1},
			"xpub6DF8uhdarytz3FWdA8TvFSvvAh8dP3283MY7p2V4SeE2wyWmG5mg5EwVvmdMVCQcoNJxGoWaU9DCWh89LojfZ537wTfunKau47EL2dhHKon",
			"xprv9zFnWC6h2cLgpmSA46vutJzBcfJ8yaJGg8cX1e5StJh45BBciYTRXSd25UEPVuesF9yog62tGAQtHjXajPPdbRCHuWS6T8XA2ECKADdw4Ef",
		},
		{
			"test vector 2 chain m/0/2147483647H/1/2147483646H",
			"fffcf9f6f3f0edeae7e4e1dedbd8d5d2cfccc9c6c3c0bdbab7b4b1aeaba8a5a29f9c999693908d8a8784817e7b7875726f6c696663605d5a5754514e4b484542",
			[]uint32{0, extkeys.HardenedKeyStart + 2147483647, 1, extkeys.HardenedKeyStart + 2147483646},
			"xpub6ERApfZwUNrhLCkDtcHTcxd75RbzS1ed54G1LkBUHQVHQKqhMkhgbmJbZRkrgZw4koxb5JaHWkY4ALHY2grBGRjaDMzQLcgJvLJuZZvRcEL",
			"xprvA1RpRA33e1JQ7ifknakTFpgNXPmW2YvmhqLQYMmrj4xJXXWYpDPS3xz7iAxn8L39njGVyuoseXzU6rcxFLJ8HFsTjSyQbLYnMpCqE2VbFWc",
		},
		{
			"test vector 2 chain m/0/2147483647H/1/2147483646H/2",
			"fffcf9f6f3f0edeae7e4e1dedbd8d5d2cfccc9c6c3c0bdbab7b4b1aeaba8a5a29f9c999693908d8a8784817e7b7875726f6c696663605d5a5754514e4b484542",
			[]uint32{0, extkeys.HardenedKeyStart + 2147483647, 1, extkeys.HardenedKeyStart + 2147483646, 2},
			"xpub6FnCn6nSzZAw5Tw7cgR9bi15UV96gLZhjDstkXXxvCLsUXBGXPdSnLFbdpq8p9HmGsApME5hQTZ3emM2rnY5agb9rXpVGyy3bdW6EEgAtqt",
			"xprvA2nrNbFZABcdryreWet9Ea4LvTJcGsqrMzxHx98MMrotbir7yrKCEXw7nadnHM8Dq38EGfSh6dqA9QWTyefMLEcBYJUuekgW4BYPJcr9E7j",
		},
	}

tests:
	for i, test := range tests {
		seed, err := hex.DecodeString(test.seed)
		if err != nil {
			t.Errorf("DecodeString #%d (%s): %v", i, test.name, err)
			continue
		}

		extKey, err := extkeys.NewMaster(seed, []byte("Bitcoin seed"))
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
		testVec1MasterPrivKey := "xprv9s21ZrQH143K3QTDL4LXw2F7HEK3wJUD2nW2nRk4stbPy6cq3jPPqjiChkVvvNKmPGJxWUtg6LnF5kejMRNNU3TGtRBeJgk33yuGBxrMPHi"
		testVec2MasterPrivKey := "xprv9s21ZrQH143K31xYSDQpPDxsXRTUcvj2iNHm5NUtrGiGG5e2DtALGdso3pGz6ssrdK4PFmM8NSpSBHNqPqm55Qn3LqFtT2emdEXVYsCzC2U"

		return []testCase{
			// Test vector 1
			{
				name:    "test vector 1 chain m",
				master:  testVec1MasterPrivKey,
				path:    []uint32{},
				wantKey: "xprv9s21ZrQH143K3QTDL4LXw2F7HEK3wJUD2nW2nRk4stbPy6cq3jPPqjiChkVvvNKmPGJxWUtg6LnF5kejMRNNU3TGtRBeJgk33yuGBxrMPHi",
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
				wantKey: "xprv9s21ZrQH143K31xYSDQpPDxsXRTUcvj2iNHm5NUtrGiGG5e2DtALGdso3pGz6ssrdK4PFmM8NSpSBHNqPqm55Qn3LqFtT2emdEXVYsCzC2U",
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
			extKey, err := extkeys.NewKeyFromString(test.master)
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
	_, err := extkeys.NewMaster(bytes.Repeat([]byte{0x00}, 15), []byte{0x00})
	if err != extkeys.ErrInvalidSeedLen {
		t.Errorf("NewMaster: mismatched error -- got: %v, want: %v",
			err, extkeys.ErrInvalidSeedLen)
	}

	// Should get an error when seed has too many bytes.
	_, err = extkeys.NewMaster(bytes.Repeat([]byte{0x00}, 65), []byte{0x00})
	if err != extkeys.ErrInvalidSeedLen {
		t.Errorf("NewMaster: mismatched error -- got: %v, want: %v",
			err, extkeys.ErrInvalidSeedLen)
	}

	// Generate a new key and neuter it to a public extended key.
	mnemonic := extkeys.NewMnemonic(extkeys.Salt)

	phrase, err := mnemonic.MnemonicPhrase(128, extkeys.EnglishLanguage)
	if err != nil {
		t.Errorf("Test failed: could not create seed: %s", err)
	}

	password := "badpassword"
	extKey, err := extkeys.NewMaster(mnemonic.MnemonicSeed(phrase, password), []byte(extkeys.Salt))
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
	_, err = pubKey.Child(extkeys.HardenedKeyStart)
	if err != extkeys.ErrDerivingHardenedFromPublic {
		t.Errorf("Child: mismatched error -- got: %v, want: %v", err, extkeys.ErrDerivingHardenedFromPublic)
	}

	_, err = pubKey.BIP44Child(extkeys.CoinTypeETH, 0)
	if err != extkeys.ErrInvalidMasterKey {
		t.Errorf("BIP44Child: mistmatched error -- got: %v, want: %v", err, extkeys.ErrInvalidMasterKey)
	}

	childKey, _ := extKey.Child(extkeys.HardenedKeyStart + 1)
	_, err = childKey.BIP44Child(extkeys.CoinTypeETH, 0) // this should be called from master only
	if err != extkeys.ErrInvalidMasterKey {
		t.Errorf("BIP44Child: mistmatched error -- got: %v, want: %v", err, extkeys.ErrInvalidMasterKey)
	}

	// NewKeyFromString failure tests.
	tests := []struct {
		name      string
		key       string
		err       error
		neuter    bool
		neuterErr error
	}{
		{
			name: "invalid key length",
			key:  "xpub1234",
			err:  extkeys.ErrInvalidKeyLen,
		},
		{
			name: "bad checksum",
			key:  "xpub661MyMwAqRbcFtXgS5sYJABqqG9YLmC4Q1Rdap9gSE8NqtwybGhePY2gZ29ESFjqJoCu1Rupje8YtGqsefD265TMg7usUDFdp6W1EBygr15",
			err:  extkeys.ErrBadChecksum,
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
	}

	for i, test := range tests {
		extKey, err := extkeys.NewKeyFromString(test.key)
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
	}
}

func TestBIP44ChildDerivation(t *testing.T) {
	keyString := "xprv9s21ZrQH143K3QTDL4LXw2F7HEK3wJUD2nW2nRk4stbPy6cq3jPPqjiChkVvvNKmPGJxWUtg6LnF5kejMRNNU3TGtRBeJgk33yuGBxrMPHi"
	derivedKey1String := "xprvA38t8tFW4vbuB7WJXEqMFmZqRrcZUKWqqMcGjjKjr2hbfvPhRtLLJGL4ayWG8shF1VkuUikVGodGshLiKRS7WrdsrGSVDQCY33qoPBxG2Kp"
	derivedKey2String := "xprvA38t8tFW4vbuDgBNpekPnuMSfpWziDLdF7W9Zd3mPy6eDEkM5F17vk59RtVoFbNdBBq84EJf5CqdZhhEoBkAM4DXHQsDqvUxVnncfnDQEFg"

	extKey, err := extkeys.NewKeyFromString(keyString)
	if err != nil {
		t.Error("NewKeyFromString: cannot create extended key")
	}

	accounKey1, err := extKey.BIP44Child(extkeys.CoinTypeETH, 0)
	if err != nil {
		t.Error("Error dering BIP44-compliant key")
	}
	if accounKey1.String() != derivedKey1String {
		t.Errorf("BIP44Child: key mismatch -- got: %v, want: %v", accounKey1.String(), derivedKey1String)
	}
	t.Logf("Account 1 key: %s", accounKey1.String())

	accounKey2, err := extKey.BIP44Child(extkeys.CoinTypeETH, 1)
	if err != nil {
		t.Error("Error dering BIP44-compliant key")
	}
	if accounKey2.String() != derivedKey2String {
		t.Errorf("BIP44Child: key mismatch -- got: %v, want: %v", accounKey2.String(), derivedKey2String)
	}
	t.Logf("Account 1 key: %s", accounKey2.String())
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
//	key, err := NewMaster(mnemonic.MnemonicSeed(phrase, password), []byte(mnemonic.salt))
//	if err != nil {
//		t.Error(err)
//	}
//	t.Logf("%x", key.KeyData)
//}
