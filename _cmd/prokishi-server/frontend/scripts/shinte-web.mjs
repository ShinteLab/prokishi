// 共有フロントパッケージ @shinte/web（core/web）を shinte-web/ にコピーする。
//
// 版は _cmd/prokishi-server/go.mod の github.com/ShinteLab/core で決まる（tools.go）。
// go にモジュールの場所を聞くので、タグならモジュールキャッシュ、
// go.mod に replace を書いていればその場所（手元の core）から取る。
// vite の alias と tsconfig の paths は shinte-web/ を見る。
import { execFileSync } from "node:child_process";
import { mkdirSync, readdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const frontend = dirname(dirname(fileURLToPath(import.meta.url)));
const appModule = dirname(frontend);
const modulePath = "github.com/ShinteLab/core";

const go = (...args) => execFileSync("go", args, { cwd: appModule, encoding: "utf8" }).trim();

go("mod", "download", modulePath);
const src = join(go("list", "-m", "-f", "{{.Dir}}", modulePath), "web");
const dst = join(frontend, "shinte-web");

rmSync(dst, { recursive: true, force: true });
mkdirSync(dst);
// モジュールキャッシュのファイルは読み取り専用なので、属性ごとコピーせず中身だけ書く
for (const name of readdirSync(src)) {
  if (/\.(js|d\.ts|json|txt)$/.test(name)) {
    writeFileSync(join(dst, name), readFileSync(join(src, name)));
  }
}
console.log(`@shinte/web: ${src}`);
