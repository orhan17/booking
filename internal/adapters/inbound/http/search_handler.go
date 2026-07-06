package http

import "net/http"

func (h *handlers) handleSearch(w http.ResponseWriter, r *http.Request) {
	var req searchRequest
	if err := decodeJSON(w, r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, err)
		return
	}

	offers, err := h.search.Search(r.Context(), req.toCriteria())
	if err != nil {
		h.writeError(w, statusFor(err), err)
		return
	}

	h.writeJSON(w, http.StatusOK, searchResponse{Offers: offersFromDomain(offers)})
}
