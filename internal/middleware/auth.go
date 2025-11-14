package middleware

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"instant-hdr-backend/internal/config"
)

const UserIDKey = "user_id"

func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := strings.TrimSpace(parts[1])
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "empty token"})
			c.Abort()
			return
		}

		// Try URL decoding in case the token was URL-encoded
		decodedToken, err := url.QueryUnescape(tokenString)
		if err == nil && decodedToken != tokenString {
			tokenString = decodedToken
		}

		// Check if token has the correct JWT format (3 parts separated by dots)
		tokenParts := strings.Split(tokenString, ".")
		if len(tokenParts) != 3 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "invalid token format",
				"message": "JWT token must have 3 parts separated by dots",
			})
			c.Abort()
			return
		}

		// Try to decode the payload to check if it's valid base64 and JSON
		// This helps diagnose encoding issues before attempting full JWT parsing
		var decodedPayload []byte
		var decodeErr error
		
		// Try RawURLEncoding first (standard for JWT)
		decodedPayload, decodeErr = base64.RawURLEncoding.DecodeString(tokenParts[1])
		if decodeErr != nil {
			// Try with padding
			payloadWithPadding := tokenParts[1]
			if len(payloadWithPadding)%4 != 0 {
				payloadWithPadding += strings.Repeat("=", 4-len(payloadWithPadding)%4)
			}
			decodedPayload, decodeErr = base64.RawURLEncoding.DecodeString(payloadWithPadding)
			if decodeErr != nil {
				// Try standard base64 encoding as last resort
				decodedPayload, decodeErr = base64.StdEncoding.DecodeString(tokenParts[1])
				if decodeErr != nil {
					c.JSON(http.StatusUnauthorized, gin.H{
						"error":   "invalid token encoding",
						"message": "token payload is not valid base64: " + decodeErr.Error(),
					})
					c.Abort()
					return
				}
			}
		}

		// Check if decoded payload is valid JSON by attempting to unmarshal
		var testClaims map[string]interface{}
		if err := json.Unmarshal(decodedPayload, &testClaims); err != nil {
			// Check for common issues
			payloadStr := string(decodedPayload)
			errorMsg := "token payload is not valid JSON. This usually means the token is corrupted, truncated, or not a valid Supabase JWT token."
			
			// Check for common malformation patterns
			if strings.Contains(payloadStr, `"exp":`) {
				// Try to find the exp value and check if it has invalid characters
				expIndex := strings.Index(payloadStr, `"exp":`)
				if expIndex != -1 {
					expValueStart := expIndex + 6 // After "exp":
					expValueEnd := strings.IndexAny(payloadStr[expValueStart:], ",}")
					if expValueEnd != -1 {
						expValue := strings.TrimSpace(payloadStr[expValueStart : expValueStart+expValueEnd])
						// Check if exp value contains non-numeric characters
						if strings.ContainsAny(expValue, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz") {
							errorMsg = "token payload contains invalid 'exp' (expiration) value with non-numeric characters. The token appears to be corrupted. Please get a fresh token from Supabase Auth."
						}
					}
				}
			}
			
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "invalid token payload",
				"message": errorMsg,
				"detail":  "JSON parse error: " + err.Error(),
			})
			c.Abort()
			return
		}

		// First, parse without verification to check token structure
		parser := jwt.NewParser()
		unverifiedToken, _, err := parser.ParseUnverified(tokenString, jwt.MapClaims{})
		if err != nil {
			// Provide more detailed error information
			errorMsg := err.Error()
			if strings.Contains(errorMsg, "could not JSON decode") {
				errorMsg = "token payload contains invalid JSON - the token may be corrupted or not a valid Supabase JWT token. Please ensure you're using a fresh token from Supabase Auth."
			}
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "invalid token structure",
				"message": errorMsg,
			})
			c.Abort()
			return
		}

		// Verify the signing method matches what Supabase uses (HS256)
		if unverifiedToken.Method.Alg() != "HS256" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "invalid token algorithm",
				"message": "token must use HS256 algorithm, got: " + unverifiedToken.Method.Alg(),
			})
			c.Abort()
			return
		}

		// Now parse and validate with signature verification
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Verify signing method - Supabase uses HS256 (HMAC)
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			if cfg.SupabaseJWTSecret == "" {
				return nil, jwt.ErrSignatureInvalid
			}
			// Supabase JWT secret is used directly as the signing key
			return []byte(cfg.SupabaseJWTSecret), nil
		}, jwt.WithValidMethods([]string{"HS256"}))

		if err != nil {
			// Provide more helpful error messages
			var errorMsg string
			if strings.Contains(err.Error(), "signature is invalid") {
				errorMsg = "token signature is invalid - check JWT secret"
			} else if strings.Contains(err.Error(), "token is expired") {
				errorMsg = "token has expired"
			} else if strings.Contains(err.Error(), "could not JSON decode") {
				errorMsg = "token is malformed - ensure you're using a valid Supabase JWT token"
			} else {
				errorMsg = err.Error()
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token", "message": errorMsg})
			c.Abort()
			return
		}

		if !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// Extract claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			c.Abort()
			return
		}

		// Extract user_id from "sub" claim
		sub, ok := claims["sub"].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user id in token"})
			c.Abort()
			return
		}

		// Store user_id in context
		c.Set(UserIDKey, sub)
		c.Next()
	}
}
