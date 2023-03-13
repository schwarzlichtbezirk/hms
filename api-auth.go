package hms

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/xml"
	"math/rand"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
)

func pubkeyAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Key [32]byte `json:"key" yaml:"key,flow" xml:"key"`
	}
	if _, err = rand.Read(ret.Key[:]); err != nil {
		WriteError500(w, r, err, AECpubkeyrand)
		return
	}

	var cell TempCell[struct{}]
	cell.Data = nil
	cell.Wait = time.AfterFunc(15*time.Second, func() {
		pubkcache.Remove(ret.Key)
	})
	pubkcache.Set(ret.Key, cell)

	WriteOK(w, r, &ret)
}

func signinAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Name string   `json:"name" yaml:"name" xml:"name"`
		PubK [32]byte `json:"pubk" yaml:"pubk" xml:"pubk"`
		Hash [32]byte `json:"hash" yaml:"hash" xml:"hash"`
	}
	var res Tokens

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if arg.Name == "" || bytes.Equal(arg.PubK[:], zero32[:]) || bytes.Equal(arg.Hash[:], zero32[:]) {
		WriteError400(w, r, ErrNoData, AECsigninnodata)
		return
	}

	var prf *Profile
	if prf = prflist.ByLogin(arg.Name); prf == nil {
		WriteError(w, r, http.StatusForbidden, ErrNoAcc, AECsigninnoacc)
		return
	}

	if _, ok := pubkcache.Get(arg.PubK); !ok {
		WriteError(w, r, http.StatusForbidden, ErrNoPubKey, AECsigninpkey)
		return
	}

	var mac = hmac.New(sha256.New, arg.PubK[:])
	mac.Write(s2b(prf.Password))
	var cmp = mac.Sum(nil)
	if !hmac.Equal(arg.Hash[:], cmp) {
		WriteError(w, r, http.StatusForbidden, ErrBadPass, AECsignindeny)
		return
	}

	pubkcache.Remove(arg.PubK) // remove public key on successful login

	res.Make(prf.ID)

	WriteOK(w, r, &res)
}

func refrshAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg Tokens
	var res Tokens

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if len(arg.Refrsh) == 0 {
		WriteError400(w, r, ErrNoData, AECrefrshnodata)
		return
	}

	var claims Claims
	if _, err = jwt.ParseWithClaims(arg.Refrsh, &claims, func(token *jwt.Token) (any, error) {
		return s2b(cfg.RefreshKey), nil
	}); err != nil {
		WriteError400(w, r, err, AECrefrshparse)
		return
	}

	res.Make(claims.UID)

	WriteOK(w, r, &res)
}

// The End.
