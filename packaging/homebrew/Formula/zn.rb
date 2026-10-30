class Zn < Formula
  desc "ZenNotes command-line interface and terminal app for Markdown notes"
  homepage "https://github.com/ZenNotes/tui"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ZenNotes/tui/releases/download/v0.13.0/zn_0.13.0_darwin_arm64.tar.gz"
      sha256 "7f36d799066abba575d58cd1821a9cfbdd997003203de28aa884d92692453c50"
    end
    on_intel do
      url "https://github.com/ZenNotes/tui/releases/download/v0.13.0/zn_0.13.0_darwin_amd64.tar.gz"
      sha256 "28ff881df056a7911674162e0f9739ce1b0405623b694746adf0619229bb9ff8"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ZenNotes/tui/releases/download/v0.13.0/zn_0.13.0_linux_arm64.tar.gz"
      sha256 "848163c70d7aa8228936e8ee876e858af4fba67d62e4d1909a985e9f8e5da730"
    end
    on_intel do
      url "https://github.com/ZenNotes/tui/releases/download/v0.13.0/zn_0.13.0_linux_amd64.tar.gz"
      sha256 "01ec73fdd4637490bd9d053f103bc49fe18e6657ac1945737b80ed0504032b74"
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
