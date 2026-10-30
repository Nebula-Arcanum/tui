class Zn < Formula
  desc "ZenNotes command-line interface and terminal app for Markdown notes"
  homepage "https://github.com/ZenNotes/tui"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ZenNotes/tui/releases/download/v0.13.1/zn_0.13.1_darwin_arm64.tar.gz"
      sha256 "f3c9f9ec75e93a7ce975e8acb099cec5c4889f38ab2e6a39a2159c7132e62d37"
    end
    on_intel do
      url "https://github.com/ZenNotes/tui/releases/download/v0.13.1/zn_0.13.1_darwin_amd64.tar.gz"
      sha256 "3aac2fb65558760a861a10a9caa8eab892d34cbdcf52230e83aa80ba6737ef04"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ZenNotes/tui/releases/download/v0.13.1/zn_0.13.1_linux_arm64.tar.gz"
      sha256 "90c3aa7463a1de2d106b31b20474ac753d4c4747a52539a753c2973b492cf448"
    end
    on_intel do
      url "https://github.com/ZenNotes/tui/releases/download/v0.13.1/zn_0.13.1_linux_amd64.tar.gz"
      sha256 "3ea08b4728c36384e8ccb008e29daa8b7ebfeb4c5264497966d41039fb326aa1"
    end
  end

  def install
    bin.install "zn"
  end

  def caveats
    <<~EOS
      zn renders display math with typst by default. To draw LaTeX math instead
      (math = "latex" under [terminal] in config.toml), zn runs pdftex and
      pdftocairo installed on your computer; it does not include TeX. On macOS:
        brew install --cask basictex
        brew install poppler
        sudo tlmgr install standalone varwidth preview xcolor amsmath amsfonts mhchem pgf
      On Linux, install your distribution's TeX Live and poppler packages.
      Run zn doctor to check; it prints the commands for your system.
    EOS
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
