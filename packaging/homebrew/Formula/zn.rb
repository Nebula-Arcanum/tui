class Zn < Formula
  desc "ZenNotes command-line interface and terminal app for Markdown notes"
  homepage "https://github.com/ZenNotes/tui"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ZenNotes/tui/releases/download/v0.8.1/zn_0.8.1_darwin_arm64.tar.gz"
      sha256 "91ca4412e5aabdee29558d1aceb94536e28c0cb616d6c8ce12e971f3b4c67222"
    end
    on_intel do
      url "https://github.com/ZenNotes/tui/releases/download/v0.8.1/zn_0.8.1_darwin_amd64.tar.gz"
      sha256 "162219ee4b0932510aaaa0f210dc65fdff70bac3f4278b8fe756748e1f164b2b"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ZenNotes/tui/releases/download/v0.8.1/zn_0.8.1_linux_arm64.tar.gz"
      sha256 "382c951f48b2f8716d69169e7468875dff3d407953667b982ca9f7d9fd0b125b"
    end
    on_intel do
      url "https://github.com/ZenNotes/tui/releases/download/v0.8.1/zn_0.8.1_linux_amd64.tar.gz"
      sha256 "fe3dce36148ea0ff0fb07615ff7f6f268ca185ddc1aff1dc43484d6ba7e8f3db"
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
