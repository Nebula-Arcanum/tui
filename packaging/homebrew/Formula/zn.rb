class Zn < Formula
  desc "ZenNotes command-line interface and terminal app for Markdown notes"
  homepage "https://github.com/ZenNotes/tui"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ZenNotes/tui/releases/download/v0.7.0/zn_0.7.0_darwin_arm64.tar.gz"
      sha256 "6272909db88987a6fda33f7e9686067ee15b514892ebda016d66f2b07af6e957"
    end
    on_intel do
      url "https://github.com/ZenNotes/tui/releases/download/v0.7.0/zn_0.7.0_darwin_amd64.tar.gz"
      sha256 "bd7292fb6b24e81afc54fd5af3958aa4022f224cca3e0cba34208b306de4dd65"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ZenNotes/tui/releases/download/v0.7.0/zn_0.7.0_linux_arm64.tar.gz"
      sha256 "943ca6a123c8e4a4fdd56f08e2fd8095b614e60ef6cfb0defe4b292b83ab3666"
    end
    on_intel do
      url "https://github.com/ZenNotes/tui/releases/download/v0.7.0/zn_0.7.0_linux_amd64.tar.gz"
      sha256 "50d794e67125126fd176e40a49a0abc17fd0cd5888ddab609bad0d6af22ccc55"
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
