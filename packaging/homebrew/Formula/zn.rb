class Zn < Formula
  desc "ZenNotes command-line interface and terminal app for Markdown notes"
  homepage "https://github.com/ZenNotes/tui"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ZenNotes/tui/releases/download/v0.8.0/zn_0.8.0_darwin_arm64.tar.gz"
      sha256 "29ebd83447c45a8fd5f8196f2c6680521bc1a2bbbec6afa609fadec480006042"
    end
    on_intel do
      url "https://github.com/ZenNotes/tui/releases/download/v0.8.0/zn_0.8.0_darwin_amd64.tar.gz"
      sha256 "7d6e91766664d72bb938e20644ab2501d8691aba4ae82168f7d18741850ef22e"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ZenNotes/tui/releases/download/v0.8.0/zn_0.8.0_linux_arm64.tar.gz"
      sha256 "b73d9356b65a597364ba09bcb086be1b37b24c7c4cd2e806e990b00e8a997f5f"
    end
    on_intel do
      url "https://github.com/ZenNotes/tui/releases/download/v0.8.0/zn_0.8.0_linux_amd64.tar.gz"
      sha256 "5f5484434dda2c4745529f4c54c955c678219ce05880e6ff3c25a5af8845b2b1"
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
