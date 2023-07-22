package web

import (
	"fmt"
	"github.com/Cloudbox/crs/util"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (c *Client) Delete(g *gin.Context) {
	// parse query
	b := new(fileRequest)
	if err := g.ShouldBindUri(b); err != nil {
		g.AbortWithError(http.StatusBadRequest, fmt.Errorf("bind uri: %w", err))
		return
	}

	b.Hash = filepath.Base(b.Hash)
	b.Filename = filepath.Base(strings.ToLower(b.Filename))
	b.Directory = filepath.Join(c.uploadDirectory, b.Hash)
	b.Filepath = filepath.Join(b.Directory, b.Filename)

	// validate request
	if !util.StringListContains(c.allowedFiles, b.Filename) {
		g.AbortWithError(http.StatusBadRequest, fmt.Errorf("file unsupported: %v", b.Filename))
		return
	}

	// validate file
	if _, err := os.Stat(b.Filepath); err != nil {
		if os.IsNotExist(err) {
			g.AbortWithError(http.StatusNotFound, fmt.Errorf("file not found: %v", b.Filename))
			return
		}

		g.AbortWithError(http.StatusInternalServerError, fmt.Errorf("file stat: %w", err))
		return
	}

	// delete file
	if err := os.Remove(b.Filepath); err != nil {
		g.AbortWithError(http.StatusInternalServerError, fmt.Errorf("file delete: %w", err))
		return
	}

	g.JSON(http.StatusOK, &fileResponse{
		Message: fmt.Sprintf("Deleted %v", b.Filename),
		Error:   false,
	})
}

func (c *Client) Purge(g *gin.Context) {
	// parse query
	b := new(purgeRequest)
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
		Message: fmt.Sprintf("Purged directory with hash %v", b.Hash),
		Error:   false,
	})
}
