import { useState } from "react";
import { getApiKey, setApiKey } from "../api/client";

export function ApiKeyBar() {
  const [value, setValue] = useState(getApiKey());
  const [saved, setSaved] = useState(false);

  function handleSave() {
    setApiKey(value.trim());
    setSaved(true);
    setTimeout(() => setSaved(false), 1500);
  }

  return (
    <div className="api-key-bar">
      <label htmlFor="api-key-input">X-API-Key</label>
      <input
        id="api-key-input"
        type="password"
        value={value}
        onChange={(e) => setValue(e.target.value)}
        placeholder="paste API key from permissions.json"
      />
      <button onClick={handleSave}>{saved ? "Saved" : "Save"}</button>
    </div>
  );
}
