package main

import (
	"fmt"
	"math/big"
	"math/rand"
	"time"
)

// Polynomial evaluation
func poly(x int64, coeffs []*big.Int) *big.Int {
	result := big.NewInt(0)
	xVal := big.NewInt(x)
	for i, coeff := range coeffs {
		term := new(big.Int).Exp(xVal, big.NewInt(int64(i)), nil)
		term.Mul(term, coeff)
		result.Add(result, term)
	}
	return result
}

// Generate shares
func makeShares(secret int64, n, k int) [][2]*big.Int {
	rand.Seed(time.Now().UnixNano())
	coeffs := make([]*big.Int, k)
	coeffs[0] = big.NewInt(secret)
	for i := 1; i < k; i++ {
		coeffs[i] = big.NewInt(rand.Int63n(100) + 1)
	}

	shares := make([][2]*big.Int, n)
	for i := 1; i <= n; i++ {
		x := big.NewInt(int64(i))
		shares[i-1] = [2]*big.Int{x, poly(x.Int64(), coeffs)}
	}
	return shares
}

// Lagrange interpolation
func interpolate(points [][2]*big.Int) *big.Int {
	result := big.NewInt(0)
	for j, pj := range points {
		xj, yj := pj[0], pj[1]
		numerator := big.NewInt(1)
		denominator := big.NewInt(1)
		for m, pm := range points {
			if j != m {
				xm := pm[0]
				numerator.Mul(numerator, new(big.Int).Neg(xm))
				denominator.Mul(denominator, new(big.Int).Sub(xj, xm))
			}
		}
		term := new(big.Int).Mul(yj, numerator)
		term.Mul(term, new(big.Int).ModInverse(denominator, nil))
		result.Add(result, term)
	}
	return result
}

// Voting mechanism
func vote(votes []int, threshold int) bool {
	sum := 0
	for _, v := range votes {
		sum += v
	}
	return sum >= threshold
}

func main() {
	// Parameters
	s, r, p := 5, 3, 4 // Starting members, replace threshold, add threshold
	m := 0.5           // Remove ratio
	secret := int64(42)

	// Initial shares
	shares := makeShares(secret, s, r)

	// Example vote results
	replaceVote := []int{1, 1, 1, 0, 0} // Replace member vote
	addVote := []int{1, 1, 1, 1, 0}     // Add member vote
	removeVote := []int{1, 1, 0, 0, 0}  // Remove member vote (for m = 0.5, at least 50% needed)

	// Replace member if vote passes
	if vote(replaceVote, r) {
		newMemberShare := makeShares(secret, 1, r)[0]
		shares = append(shares[1:], newMemberShare)
	}

	// Add member if vote passes
	if vote(addVote, p) {
		newMemberShare := makeShares(secret, 1, r)[0]
		shares = append(shares, newMemberShare)
	}

	// Remove member if vote passes
	if vote(removeVote, int(m*float64(s))) {
		shares = shares[:len(shares)-1]
	}

	// Reconstruct secret
	reconstructedSecret := interpolate(shares[:r])
	fmt.Printf("Reconstructed Secret: %d\n", reconstructedSecret)
}
