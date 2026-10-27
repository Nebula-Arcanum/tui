class Zn < Formula
  desc "ZenNotes command-line interface and terminal app for Markdown notes"
  homepage "https://github.com/ZenNotes/tui"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ZenNotes/tui/releases/download/v0.12.0/zn_0.12.0_darwin_arm64.tar.gz"
      sha256 "9928e735c6f30e703b84436559cf2740905f84df9fc873242561d983c8c152b7"
    end
    on_intel do
      url "https://github.com/ZenNotes/tui/releases/download/v0.12.0/zn_0.12.0_darwin_amd64.tar.gz"
      sha256 "6f963a7581328e315c94bce791787922bef09b4dc9c6007b2ff0e4b948f51e95"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ZenNotes/tui/releases/download/v0.12.0/zn_0.12.0_linux_arm64.tar.gz"
      sha256 "b93e5a3c67bd7e40672abda3246e4fdebd2e6bba64a4b29f722f9f0855b4a62c"
    end
    on_intel do
      url "https://github.com/ZenNotes/tui/releases/download/v0.12.0/zn_0.12.0_linux_amd64.tar.gz"
      sha256 "f6c1ac2260ae91d414d5118efee37972418b5df146f7e0135879c4955f8ed693"
    end
  end

  def install
    bin.install "zn"
  end

  def caveats
    <<~EOS
      zn renders display math with typst by default. With math = "latex" under
      [terminal] in config.toml, it needs pdftex and pdftocairo on PATH. A
      basic TeX Live is enough, plus poppler and the packages zn uses:
        brew install poppler
        tlmgr install standalone varwidth preview xcolor amsmath amsfonts mhchem pgf
      Add other packages with tlmgr later, as your preamble.tex needs them.
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
