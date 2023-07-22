package web

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"path/filepath"
)

type deleteRequest struct {
	Hash      string `uri:"hash" binding:"required"`
	Directory string
}

func (c *Client) Delete(g *gin.Context) {
	// parse query
	b := new(deleteRequest)
	if err := g.ShouldBindUri(b); err != nil {
		g.AbortWithError(http.StatusBadRequest, fmt.Errorf("bind uri: %w", err))
		return
	}

	b.Hash = filepath.Base(b.Hash)
	b.Directory = filepath.Join(c.uploadDirectory, b.Hash)

	// validate directory
	if _, err := os.Stat(b.Directory); err != nil {
		if os.IsNotExist(err) {
			g.AbortWithError(http.StatusNotFound, fmt.Errorf("directory not found: %v", b.Hash))
			return
		}

		g.AbortWithError(http.StatusInternalServerError, fmt.Errorf("directory stat: %w", err))
		return
	}

	// delete directory
	if err := os.RemoveAll(b.Directory); err != nil {
		g.AbortWithError(http.StatusInternalServerError, fmt.Errorf("directory delete: %w", err))
		return
	}

	g.JSON(http.StatusOK, &fileResponse{
		Message: fmt.Sprintf("Deleted directory with hash %v", b.Hash),
		Error:   false,
	})
}
