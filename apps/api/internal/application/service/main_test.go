package service

import (
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestMain sets the Gin mode once; gin.SetMode writes package globals and must not be
// called from parallel tests.
func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}
