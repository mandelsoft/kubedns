package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk" // Used to retrieve and handle JSON Web Keys
)

// Example function to load a secure HTTP client configuration that trusts the cluster's CA
func getSecureHTTPClient() (*http.Client, error) {
	// NOTE: For a Kube client-go context, you would use rest.TransportFor(config)
	// to correctly configure the transport with the cluster's CA certificate.

	// For this example, we return a simple client (you must secure this in prod)
	return &http.Client{Timeout: 10 * time.Second}, nil
}

// GetVerifierKeyFromToken analyzes the JWT, finds the issuer, and retrieves the necessary public key.
func GetVerifierKeyFromToken(tokenString string) (jwt.Keyfunc, error) {

	// 1. Parse the token to get the Header (for KID) and Payload (for ISS)
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token header/claims: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims format")
	}

	issuer, ok := claims["iss"].(string)
	if !ok || issuer == "" {
		return nil, fmt.Errorf("token payload missing 'iss' (issuer) claim")
	}

	// 2. Derive the OIDC discovery URL from the Issuer
	// This assumes the issuer URL is the base for the standard K8s OIDC discovery endpoints.
	// In a real environment, K8s should be configured with a reachable issuer URL.
	jwksURL := fmt.Sprintf("%s/.well-known/openid-configuration", issuer)

	// 3. Use the jwk library to discover the keys
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Configure the client (essential for production to handle cluster CA trust)
	httpClient, err := getSecureHTTPClient()
	if err != nil {
		return nil, err
	}

	// Fetch the JWKS from the discovery endpoint
	// The jwk.Fetch() function handles the OIDC configuration lookup for us if pointed to the base URL
	jwks, err := jwk.Fetch(ctx, jwksURL, jwk.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS from %s: %w", jwksURL, err)
	}

	// 4. Return a Keyfunc for the JWT library
	// The jwt library needs a function that takes the token and returns the verification key.
	return func(token *jwt.Token) (interface{}, error) {
		// Use the token's "kid" header to find the specific key in the set.
		keyID, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("token header missing 'kid' claim required for validation")
		}

		if key, ok := jwks.LookupKeyID(keyID); ok {
			// Convert the JWK (JSON Web Key) interface to a standard crypto key (e.g., *rsa.PublicKey)
			var pubKey interface{}
			if err := key.Raw(&pubKey); err != nil {
				return nil, fmt.Errorf("failed to convert JWK to public key interface: %w", err)
			}
			return pubKey, nil
		}

		return nil, fmt.Errorf("key ID %s not found in JWKS", keyID)
	}, nil
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <token>\n", os.Args[0])
		os.Exit(1)
	}
	// Placeholder for a token acquired earlier
	// NOTE: This MUST be a real, signed token string for parsing to work correctly.
	dummyTokenString := os.Args[1]

	keyFunc, err := GetVerifierKeyFromToken(dummyTokenString)
	if err != nil {
		fmt.Printf("Error setting up verifier key function: %v\n", err)
		return
	}

	// Now, validate the token using the returned KeyFunc
	// This function will automatically call the KeyFunc to get the key before verifying the signature.
	verifiedToken, err := jwt.Parse(dummyTokenString, keyFunc, jwt.WithLeeway(5*time.Second))

	if err != nil {
		fmt.Printf("Validation Failed: %v\n", err)
	} else if verifiedToken.Valid {
		fmt.Println("Token signature successfully validated using OIDC discovery.")
	} else {
		fmt.Println("Token is invalid for an unknown reason.")
	}
}
