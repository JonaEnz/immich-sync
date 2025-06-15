# This is an example PKGBUILD file. Use this as a start to creating your own,
# and remove these comments. For more information, see 'man PKGBUILD'.

# Maintainer: Jona Enzinger <jona_enzinger@outlook.com>
pkgname=immich-sync
pkgver=0.1.0
pkgrel=1
pkgdesc="A service to sync images from a local directory to your immich server"
arch=('x86_64')
url="https://github.com/JonaEnz/immich-sync"
license=('MIT')
makedepends=('git' 'go>=1.24.4')
source=("$pkgname-$pkgver.tar.gz::https://github.com/JonaEnz/immich-sync/archive/v${pkgver}.tar.gz")
sha256sums=('08c7e1af765d4c79c2610e59eb668308a5d9fde744d535923e877fa81d012dc3')
_commit=('eafd7051029a19a0c9ac5c9430fa7b450d0436d5')

build() {
	cd "$pkgname-$pkgver"
	go build -o ./immich-sync
	for shell in bash fish zsh; do
    ./immich-sync completion "$shell" > "$shell-completion"
  done
}

# check() {
# 	cd "$pkgname-$pkgver"
# 	make -k check
# }

package() {
	cd "$pkgname-$pkgver"
	install -Dm755 -t "$pkgdir/usr/bin" immich-sync
	mkdir -p "${pkgdir}/usr/share/bash-completion/completions/"
  mkdir -p "${pkgdir}/usr/share/zsh/site-functions/"
  mkdir -p "${pkgdir}/usr/share/fish/vendor_completions.d/"
  install -Dm644 bash-completion "$pkgdir/usr/share/bash-completion/completions/immich-sync"
  install -Dm644 fish-completion "$pkgdir/usr/share/fish/vendor_completions.d/immich-sync.fish"
  install -Dm644 zsh-completion "$pkgdir/usr/share/zsh/site-functions/_immich-sync"

  install -Dm644 -t "$pkgdir/usr/share/licenses/$pkgname" LICENSE

  install -Dm644 immich-sync.service "$pkgdir/usr/lib/systemd/immich-sync.service" 
}
