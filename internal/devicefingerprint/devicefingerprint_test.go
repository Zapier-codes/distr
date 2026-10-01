package devicefingerprint

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

var validDigest = strings.Repeat("ab", 32)

func TestValidClientHash(t *testing.T) {
	g := NewWithT(t)
	g.Expect(ValidClientHash(validDigest)).To(BeTrue())
	g.Expect(ValidClientHash("")).To(BeFalse())
	g.Expect(ValidClientHash(strings.Repeat("ab", 31))).To(BeFalse())
	g.Expect(ValidClientHash(strings.Repeat("AB", 32))).To(BeFalse())
	g.Expect(ValidClientHash(strings.Repeat("zz", 32))).To(BeFalse())
	g.Expect(ValidClientHash(validDigest + "\n")).To(BeFalse())
}

func TestHash(t *testing.T) {
	g := NewWithT(t)
	salt := strings.Repeat("s", 32)

	hash, err := Hash(salt, validDigest)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(hash).To(HaveLen(32))

	again, err := Hash(salt, validDigest)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(again).To(Equal(hash))

	otherSalt, err := Hash(strings.Repeat("t", 32), validDigest)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(otherSalt).NotTo(Equal(hash))

	otherDigest, err := Hash(salt, strings.Repeat("cd", 32))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(otherDigest).NotTo(Equal(hash))
}

func TestHashRejects(t *testing.T) {
	g := NewWithT(t)
	_, err := Hash("", validDigest)
	g.Expect(err).To(HaveOccurred())
	_, err = Hash(strings.Repeat("s", 32), "not-a-digest")
	g.Expect(err).To(MatchError(ErrInvalidClientHash))
}
