import ExecutionDetailClient from './ExecutionDetailClient';

// Required for static export of dynamic routes
export function generateStaticParams() {
  return [{ executionId: 'placeholder' }];
}

export default function ExecutionDetailPage({ params }: { params: Promise<{ executionId: string }> }) {
  // Convert params Promise to plain object for client component
  return <ExecutionDetailClient paramsPromise={params} />;
}
