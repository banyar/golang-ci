import { useLang } from "../lib/lang";

export function LangToggle() {
  const { lang, setLang } = useLang();

  return (
    <div className="lang-toggle">
      <button
        className={lang === "my" ? "lang-btn lang-btn-active" : "lang-btn"}
        onClick={() => setLang("my")}
      >
        မြန်မာ
      </button>
      <button
        className={lang === "en" ? "lang-btn lang-btn-active" : "lang-btn"}
        onClick={() => setLang("en")}
      >
        EN
      </button>
    </div>
  );
}
