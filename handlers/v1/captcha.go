package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterCaptcha register get challenge from geetest
func RegisterCaptcha(
	registerCaptcha dependencyRegisterCaptcha,
	getCaptchaID dependencyGetCaptchaID,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		challenge, err := registerCaptcha()
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		captchaID := getCaptchaID()

		c.JSON(http.StatusOK, map[string]string{
			"captcha_id": captchaID,
			"challenge":  challenge,
		})
	}
}