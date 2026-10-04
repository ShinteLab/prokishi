# prokishi の TODO

## prokishi（クライアント）

### macOS / Linux でのテスト

release.yml でクライアントを Windows / macOS / Linux 向けにビルドして配っているが、
実機で確認しているのは Windows だけ。macOS と Linux で次を確かめる。

- [ ] macOS: 配布物（`prokishi`）がそのまま起動できるか（署名していないので Gatekeeper に止められないか）
- [ ] macOS: ShogiHome にエンジンとして登録し、対局・検討ができるか
- [ ] Linux: 配布物が起動し、将棋ソフトにエンジンとして登録して使えるか
- [ ] 共通: `prokishi.ini` を実行ファイルと同じ場所から読むか。無いときに雛形がそこに作られるか
- [ ] 共通: `prokishi.log` が実行ファイルと同じ場所に出るか
- [ ] 共通: `prokishi version` でバージョンが出るか
- [ ] 共通: quit・強制終了・サーバ停止のそれぞれで、prokishi とサーバ側のエンジンがどうなるか
