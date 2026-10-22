package raw

import (
	"io"
	"net/http"

	"github.com/boubou1100/grange/internal/meta"
)

func copyTo(w http.ResponseWriter, r io.Reader) (int64, error) { return io.Copy(w, r) }

func (h *handler) delete(w http.ResponseWriter, r *http.Request, p string) {
	// Le blob n'est pas supprimé ici : il peut être partagé par d'autres assets.
	// Un job de garbage collection (à écrire) supprimera les blobs orphelins.
	_, err := h.d.Meta.DeleteAsset(r.Context(), h.repo.Name, p)
	if err == meta.ErrNotFound {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, "metadata error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
