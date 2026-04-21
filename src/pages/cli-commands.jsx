import React from 'react';
import Layout from '@theme/Layout';

export default function CliCommandsPage() {
  return (
    <Layout title="CLI Commands" description="TARINIO CLI command reference entry point">
      <main className="overview-page">
        <h1>CLI Commands</h1>
        <p>Канонический справочник по командам CLI поддерживается в отдельном документе репозитория.</p>
        <ul>
          <li>
            <a href="https://github.com/BerkutSolutions/tarinio/blob/main/docs/CLI_COMMANDS.md">
              docs/CLI_COMMANDS.md on GitHub
            </a>
          </li>
        </ul>
        <p>
          Эта точка входа нужна, чтобы русская и английская ветки документации вели к одному
          стабильному маршруту внутри портала.
        </p>
      </main>
    </Layout>
  );
}
