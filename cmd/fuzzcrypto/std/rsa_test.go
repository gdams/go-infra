// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package std_test

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"errors"
	"math/big"
	"strings"
	"testing"
)

var rsaKeySeeds = []struct {
	e             int
	n, d, p, q, r string
}{
	{
		3,
		"14314132931241006650998084889274020608918049032671858325988396851334124245188214251956198731333464217832226406088020736932173064754214329009979944037640912127943488972644697423190955557435910767690712778463524983667852819010259499695177313115447116110358524558307947613422897787329221478860907963827160223559690523660574329011927531289655711860504630573766609239332569210831325633840174683944553667352219670930408593321661375473885147973879086994006440025257225431977751512374815915392249179976902953721486040787792801849818254465486633791826766873076617116727073077821584676715609985777563958286637185868165868520557",
		"9542755287494004433998723259516013739278699355114572217325597900889416163458809501304132487555642811888150937392013824621448709836142886006653296025093941418628992648429798282127303704957273845127141852309016655778568546006839666463451542076964744073572349705538631742281931858219480985907271975884773482372966847639853897890615456605598071088189838676728836833012254065983259638538107719766738032720239892094196108713378822882383694456030043492571063441943847195939549773271694647657549658603365629458610273821292232646334717612674519997533901052790334279661754176490593041941863932308687197618671528035670452762731",
		"130903255182996722426771613606077755295583329135067340152947172868415809027537376306193179624298874215608270802054347609836776473930072411958753044562214537013874103802006369634761074377213995983876788718033850153719421695468704276694983032644416930879093914927146648402139231293035971427838068945045019075433",
		"109348945610485453577574767652527472924289229538286649661240938988020367005475727988253438647560958573506159449538793540472829815903949343191091817779240101054552748665267574271163617694640513549693841337820602726596756351006149518830932261246698766355347898158548465400674856021497190430791824869615170301029",
		"",
	},
	{
		3,
		"16346378922382193400538269749936049106320265317511766357599732575277382844051791096569333808598921852351577762718529818072849191122419410612033592401403764925096136759934497687765453905884149505175426053037420486697072448609022753683683718057795566811401938833367954642951433473337066311978821180526439641496973296037000052546108507805269279414789035461158073156772151892452251106173507240488993608650881929629163465099476849643165682709047462010581308719577053905787496296934240246311806555924593059995202856826239801816771116902778517096212527979497399966526283516447337775509777558018145573127308919204297111496233",
		"10897585948254795600358846499957366070880176878341177571733155050184921896034527397712889205732614568234385175145686545381899460748279607074689061600935843283397424506622998458510302603922766336783617368686090042765718290914099334449154829375179958369993407724946186243249568928237086215759259909861748642124071874879861299389874230489928271621259294894142840428407196932444474088857746123104978617098858619445675532587787023228852383149557470077802718705420275739737958953794088728369933811184572620857678792001136676902250566845618813972833750098806496641114644760255910789397593428910198080271317419213080834885003",
		"1025363189502892836833747188838978207017355117492483312747347695538428729137306368764177201532277413433182799108299960196606011786562992097313508180436744488171474690412562218914213688661311117337381958560443",
		"3467903426626310123395340254094941045497208049900750380025518552334536945536837294961497712862519984786362199788654739924501424784631315081391467293694361474867825728031147665777546570788493758372218019373",
		"4597024781409332673052708605078359346966325141767460991205742124888960305710298765592730135879076084498363772408626791576005136245060321874472727132746643162385746062759369754202494417496879741537284589047",
	},
}

func newRSAKey(e int, bN, bD, bP, bQ, bR string) *rsa.PrivateKey {
	fromBase10 := func(base10 string) *big.Int {
		i, ok := new(big.Int).SetString(base10, 10)
		if !ok {
			return big.NewInt(0)
		}
		return i
	}
	key := &rsa.PrivateKey{
		PublicKey: rsa.PublicKey{
			N: fromBase10(bN),
			E: e,
		},
		D:      fromBase10(bD),
		Primes: []*big.Int{fromBase10(bP), fromBase10(bQ)},
	}
	if bR != "" {
		key.Primes = append(key.Primes, fromBase10(bR))
	}
	return key
}

func FuzzRSAOAEP(f *testing.F) {
	for _, s := range rsaKeySeeds {
		f.Add(s.e, s.n, s.d, s.p, s.q, s.r, []byte("testing"), []byte("a label"))
	}
	f.Fuzz(func(t *testing.T, e int, bN, bD, bP, bQ, bR string, msg, label []byte) {
		key := newRSAKey(e, bN, bD, bP, bQ, bR)
		if key.Validate() != nil {
			return
		}
		enc, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, &key.PublicKey, msg, label)
		if err != nil {
			if !errorCorrectlyRejectedRSAInput(err) {
				t.Fatal(err)
			}
			return
		}
		dec, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, key, enc, label)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(dec, []byte(msg)) {
			t.Errorf("got:%x want:%x", dec, msg)
		}
	})
}

func FuzzRSAPKCS1(f *testing.F) {
	for _, s := range rsaKeySeeds {
		f.Add(s.e, s.n, s.d, s.p, s.q, s.r, []byte("testing"))
	}
	f.Fuzz(func(t *testing.T, e int, bN, bD, bP, bQ, bR string, msg []byte) {
		key := newRSAKey(e, bN, bD, bP, bQ, bR)
		if key.Validate() != nil {
			return
		}
		enc, err := rsa.EncryptPKCS1v15(rand.Reader, &key.PublicKey, msg)
		if err != nil {
			if !errorCorrectlyRejectedRSAInput(err) {
				t.Fatal(err)
			}
			return
		}
		dec, err := rsa.DecryptPKCS1v15(rand.Reader, key, enc)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(dec, []byte(msg)) {
			t.Errorf("got:%x want:%x", dec, msg)
		}
	})
}

func FuzzRSASignPSS(f *testing.F) {
	for _, s := range rsaKeySeeds {
		f.Add(uint8(1), s.e, s.n, s.d, s.p, s.q, s.r, []byte("testing"))
	}
	opts := rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash}
	f.Fuzz(func(t *testing.T, x uint8, e int, bN, bD, bP, bQ, bR string, msg []byte) {
		key := newRSAKey(e, bN, bD, bP, bQ, bR)
		if key.Validate() != nil {
			return
		}
		hashed := sha256.Sum256(msg)
		sig, err := rsa.SignPSS(rand.Reader, key, crypto.SHA256, hashed[:], &opts)
		if err != nil {
			if !errorCorrectlyRejectedRSAInput(err) {
				t.Fatal(err)
			}
			return
		}
		err = rsa.VerifyPSS(&key.PublicKey, crypto.SHA256, hashed[:], sig, &opts)
		if err != nil {
			t.Error(err)
		}
		hashed = xor(x, hashed)
		err = rsa.VerifyPSS(&key.PublicKey, crypto.SHA256, hashed[:], sig, &opts)
		if err == nil {
			t.Errorf("Verify succeeded despite intentionally invalid hash!")
		}
	})
}

func FuzzRSASignPKCS1v15(f *testing.F) {
	for _, s := range rsaKeySeeds {
		f.Add(uint8(1), s.e, s.n, s.d, s.p, s.q, s.r, []byte("testing"))
	}
	f.Fuzz(func(t *testing.T, x uint8, e int, bN, bD, bP, bQ, bR string, msg []byte) {
		key := newRSAKey(e, bN, bD, bP, bQ, bR)
		if key.Validate() != nil {
			return
		}
		hashed := sha256.Sum256(msg)
		sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hashed[:])
		if err != nil {
			if !errorCorrectlyRejectedRSAInput(err) {
				t.Fatal(err)
			}
			return
		}
		err = rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, hashed[:], sig)
		if err != nil {
			t.Error(err)
		}
		hashed = xor(x, hashed)
		err = rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, hashed[:], sig)
		if err == nil {
			t.Errorf("Verify succeeded despite intentionally invalid hash!")
		}
	})
}

func errorCorrectlyRejectedRSAInput(err error) bool {
	if err == nil {
		return true
	}
	// Key size being too small is an invalid input, not a problem.
	errStr := err.Error()
	switch errStr {
	case "crypto/rsa: key size too small for PSS signature", "crypto/rsa: invalid key size":
		return true
	}
	if strings.Contains(errStr, "data too large for key size") {
		// This is the OpenSSL error message for rsa.ErrMessageTooLong.
		return true
	}
	if errors.Is(err, rsa.ErrMessageTooLong) {
		return true
	}
	return false
}

// xor returns b with one byte xored with x.
// The xored byte depends on x.
func xor(x uint8, b [sha256.Size]byte) [sha256.Size]byte {
	// x can be in the [0,255) range,
	// but idx must be in the [0,sha256.Size] range.
	idx := int(x) % len(b)
	if x == 0 {
		// a 0 xor doesn't modify the byte value.
		x = 1
	}
	b[idx] ^= x
	return b
}
