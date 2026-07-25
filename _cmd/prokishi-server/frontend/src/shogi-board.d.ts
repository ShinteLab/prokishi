// 共有 Web Component <shogi-board> (@shinte/web) を JSX で使うための型宣言。
// index.js を import すると customElements に登録される(App 側で副作用 import する)。
import type * as React from "react";

declare global {
  namespace JSX {
    interface IntrinsicElements {
      "shogi-board": React.DetailedHTMLProps<
        React.HTMLAttributes<HTMLElement> & {
          sfen?: string;
          "last-move"?: string;
          flip?: boolean;
          coords?: string;
        },
        HTMLElement
      >;
      "shogi-hand": React.DetailedHTMLProps<
        React.HTMLAttributes<HTMLElement> & {
          hands?: string;
          side?: string;
          flip?: boolean;
        },
        HTMLElement
      >;
    }
  }
}
