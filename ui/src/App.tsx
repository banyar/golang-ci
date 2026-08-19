import { BrowserRouter, Link, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ApiKeyBar } from "./components/ApiKeyBar";
import { LangToggle } from "./components/LangToggle";
import { LangProvider, useLang } from "./lib/lang";
import { getStrings } from "./lib/strings";
import { Dashboard } from "./pages/Dashboard";
import { IssueList } from "./pages/IssueList";
import { PlanViewer } from "./pages/PlanViewer";
import { FixProgress } from "./pages/FixProgress";
import { History } from "./pages/History";

const queryClient = new QueryClient();

function AppShell() {
  const { lang } = useLang();
  const t = getStrings(lang);

  return (
    <BrowserRouter>
      <header className="app-header">
        <nav>
          <Link to="/">{t.nav.dashboard}</Link>
          <Link to="/history">{t.nav.history}</Link>
        </nav>
        <div className="app-header-right">
          <LangToggle />
          <ApiKeyBar />
        </div>
      </header>
      <main>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/scans/:scanId/issues" element={<IssueList />} />
          <Route path="/plans/:id" element={<PlanViewer />} />
          <Route path="/fixes/:id" element={<FixProgress />} />
          <Route path="/history" element={<History />} />
        </Routes>
      </main>
    </BrowserRouter>
  );
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <LangProvider>
        <AppShell />
      </LangProvider>
    </QueryClientProvider>
  );
}

export default App;
