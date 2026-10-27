class Zn < Formula
  desc "ZenNotes command-line interface and terminal app for Markdown notes"
  homepage "https://github.com/ZenNotes/tui"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ZenNotes/tui/releases/download/v0.12.2/zn_0.12.2_darwin_arm64.tar.gz"
      sha256 "7e7b3be038dfa63dc4a79bc6068cb459462cf81e809e5b070d73bfbe3e7bd5ac"
    end
    on_intel do
      url "https://github.com/ZenNotes/tui/releases/download/v0.12.2/zn_0.12.2_darwin_amd64.tar.gz"
      sha256 "091f8b2a2f5ceb8f03acfd90f4337b085551b36058b7326b9ee64d4b1a330a12"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ZenNotes/tui/releases/download/v0.12.2/zn_0.12.2_linux_arm64.tar.gz"
      sha256 "882ec3fb8f988069c2208412e3ce72bfe8be8dfff461c0ec503f5734c7417786"
    end
    on_intel do
      url "https://github.com/ZenNotes/tui/releases/download/v0.12.2/zn_0.12.2_linux_amd64.tar.gz"
      sha256 "2617004481aee2e4f9677cc362d76ed9c0c414e6ec7858fa34db3d09156a33fe"
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
