class Zn < Formula
  desc "ZenNotes command-line interface and terminal app for Markdown notes"
  homepage "https://github.com/ZenNotes/tui"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ZenNotes/tui/releases/download/v0.7.1/zn_0.7.1_darwin_arm64.tar.gz"
      sha256 "cb0c90410fcd86972b4e62a24840bd0f5d2d82d403cfc556399e3c735b43651c"
    end
    on_intel do
      url "https://github.com/ZenNotes/tui/releases/download/v0.7.1/zn_0.7.1_darwin_amd64.tar.gz"
      sha256 "c6df5efe8d18dfa598e1ce8698fdd1a2eccf76892d94533479838cdd9a3e5d37"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ZenNotes/tui/releases/download/v0.7.1/zn_0.7.1_linux_arm64.tar.gz"
      sha256 "1151c6daed55825e881014873802c32c056dff68797e64ed83451b9fdfe3f0b9"
    end
    on_intel do
      url "https://github.com/ZenNotes/tui/releases/download/v0.7.1/zn_0.7.1_linux_amd64.tar.gz"
      sha256 "b61d145931c32b417eaf5f8bce379e7f94e327e3b5527f7868dcba3a3a178ac2"
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
