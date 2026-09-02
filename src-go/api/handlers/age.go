package handlers

import (
	"github.com/legiz-ru/prizrak-box/internal"
	"github.com/metacubex/chi"
	"github.com/metacubex/http"
	"github.com/metacubex/chi/render"
)

func Age(r chi.Router) {
	r.Mount("/age", ageRouter())
}

func ageRouter() http.Handler {
	r := chi.NewRouter()
	r.Get("/keypair", generateKeypair)
	return r
}

type keypairResponse struct {
	SecretKey string `json:"secretKey"`
	PublicKey string `json:"publicKey"`
}

func generateKeypair(w http.ResponseWriter, r *http.Request) {
	keyType := r.URL.Query().Get("type")

	secretKey, publicKey, err := internal.GenerateAgeKeyPair(keyType)
	if err != nil {
		ErrorResponse(w, r, err)
		return
	}

	render.JSON(w, r, keypairResponse{
		SecretKey: secretKey,
		PublicKey: publicKey,
	})
}
