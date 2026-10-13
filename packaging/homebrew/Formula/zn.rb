class Zn < Formula
  desc "ZenNotes command-line interface and terminal app for Markdown notes"
  homepage "https://github.com/ZenNotes/tui"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ZenNotes/tui/releases/download/v0.7.2/zn_0.7.2_darwin_arm64.tar.gz"
      sha256 "395163f84ded70a080bfff897785a91f6919e53f490d5ab1f3057545f69c9d83"
    end
    on_intel do
      url "https://github.com/ZenNotes/tui/releases/download/v0.7.2/zn_0.7.2_darwin_amd64.tar.gz"
      sha256 "39977ccb78c88b97e39046b9b3aaf24ecc460fa24fd61085b085679c3bad79ec"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ZenNotes/tui/releases/download/v0.7.2/zn_0.7.2_linux_arm64.tar.gz"
      sha256 "3b702e1db4693bacdd60f07710aafadf835d35dd0f7d6084a933ad4a98dc1682"
    end
    on_intel do
      url "https://github.com/ZenNotes/tui/releases/download/v0.7.2/zn_0.7.2_linux_amd64.tar.gz"
      sha256 "17f2c6c7ff5b484fc132b0a333827aeaafaf42d271ec378877580647ff04fa95"
    end
  end

  def install
    bin.install "zn"
  end

  test do
    ENV["ZENNOTES_CONFIG_DIR"] = (testpath/"config").to_s
    assert_match version.to_s, shell_output("#{bin}/zn --version")

    system bin/"zn", "init", testpath/"notes"
    (testpath/"notes/Homebrew.md").write("# Homebrew\n\nA note from the formula test.\n")
    assert_equal "# Homebrew\n\nA note from the formula test.\n",
                 shell_output("#{bin}/zn read Homebrew.md --vault #{testpath}/notes")
  end
end
