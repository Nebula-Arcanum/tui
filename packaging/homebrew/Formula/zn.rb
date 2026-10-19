class Zn < Formula
  desc "ZenNotes command-line interface and terminal app for Markdown notes"
  homepage "https://github.com/ZenNotes/tui"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ZenNotes/tui/releases/download/v0.9.0/zn_0.9.0_darwin_arm64.tar.gz"
      sha256 "1f97fcbf307ccfba03496311bdc04e9708c10e822ac2abdf03ffa7131197f92e"
    end
    on_intel do
      url "https://github.com/ZenNotes/tui/releases/download/v0.9.0/zn_0.9.0_darwin_amd64.tar.gz"
      sha256 "2f95711520fda368cff318d744b87b3afa98cccefcb34b35d86cce967c4c0bb8"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ZenNotes/tui/releases/download/v0.9.0/zn_0.9.0_linux_arm64.tar.gz"
      sha256 "b0a08f2575fec89e87f76fcb5dc9a7410c9b8e3327922e587605ccdd3588f2dc"
    end
    on_intel do
      url "https://github.com/ZenNotes/tui/releases/download/v0.9.0/zn_0.9.0_linux_amd64.tar.gz"
      sha256 "6ca45da7e8adcbed0037888961239fe4dbb6b960ec87844a8b6fb2439484f523"
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
