import React from 'react';
import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight } from 'lucide-react';

interface PaginationControllerProps {
  currentPage: number;
  totalItems: number;
  pageSize: number;
  onPageChange: (page: number) => void;
  onPageSizeChange?: (size: number) => void;
  pageSizeOptions?: number[];
  itemLabel?: string;
}

export const PaginationController: React.FC<PaginationControllerProps> = ({
  currentPage,
  totalItems,
  pageSize,
  onPageChange,
  onPageSizeChange,
  pageSizeOptions = [10, 20, 50],
  itemLabel = 'opportunities',
}) => {
  const totalPages = Math.max(1, Math.ceil(totalItems / pageSize));
  const safePage = Math.min(Math.max(1, currentPage), totalPages);
  const startItem = totalItems === 0 ? 0 : (safePage - 1) * pageSize + 1;
  const endItem = Math.min(totalItems, safePage * pageSize);

  // Build compact numbered page list with ellipsis for large page counts
  const getVisiblePages = (): (number | 'ellipsis')[] => {
    if (totalPages <= 7) {
      return Array.from({ length: totalPages }, (_, i) => i + 1);
    }
    const pages: (number | 'ellipsis')[] = [1];
    if (safePage > 3) pages.push('ellipsis');
    const start = Math.max(2, safePage - 1);
    const end = Math.min(totalPages - 1, safePage + 1);
    for (let p = start; p <= end; p++) {
      pages.push(p);
    }
    if (safePage < totalPages - 2) pages.push('ellipsis');
    pages.push(totalPages);
    return pages;
  };

  if (totalItems === 0) return null;

  return (
    <div className="nexa-card rounded-3xl px-5 py-3.5 flex flex-col sm:flex-row items-center justify-between gap-3">
      <div className="flex items-center gap-3 flex-wrap justify-center sm:justify-start">
        <span className="text-xs text-nexa-text-secondary font-semibold">
          Showing{' '}
          <strong className="text-nexa-text-primary">
            {startItem}–{endItem}
          </strong>{' '}
          of <strong className="text-nexa-text-primary">{totalItems}</strong> {itemLabel}
        </span>

        {onPageSizeChange && (
          <div className="flex items-center gap-1.5">
            <span className="text-[11px] text-nexa-text-muted font-semibold">Per page:</span>
            <select
              value={pageSize}
              onChange={(e) => onPageSizeChange(Number(e.target.value))}
              className="px-2.5 py-1 rounded-full border border-nexa-border bg-nexa-bg-base text-nexa-text-primary text-xs font-bold cursor-pointer"
              aria-label="Items per page"
            >
              {pageSizeOptions.map((opt) => (
                <option key={opt} value={opt}>
                  {opt}
                </option>
              ))}
            </select>
          </div>
        )}
      </div>

      <div className="flex items-center gap-1.5">
        <button
          type="button"
          onClick={() => onPageChange(1)}
          disabled={safePage <= 1}
          title="First page"
          className="p-2 rounded-full border border-nexa-border bg-nexa-bg-base text-nexa-text-secondary hover:border-nexa-brand hover:text-nexa-brand disabled:opacity-40 disabled:pointer-events-none transition cursor-pointer"
        >
          <ChevronsLeft className="w-3.5 h-3.5" />
        </button>
        <button
          type="button"
          onClick={() => onPageChange(safePage - 1)}
          disabled={safePage <= 1}
          title="Previous page"
          className="px-3 py-1.5 rounded-full border border-nexa-border bg-nexa-bg-base text-nexa-text-secondary hover:border-nexa-brand hover:text-nexa-brand disabled:opacity-40 disabled:pointer-events-none text-xs font-bold flex items-center gap-1 transition cursor-pointer"
        >
          <ChevronLeft className="w-3.5 h-3.5" />
          <span className="hidden sm:inline">Prev</span>
        </button>

        <div className="flex items-center gap-1 px-1">
          {getVisiblePages().map((entry, idx) =>
            entry === 'ellipsis' ? (
              <span key={`e-${idx}`} className="px-1.5 text-xs text-nexa-text-muted font-bold select-none">
                …
              </span>
            ) : (
              <button
                key={entry}
                type="button"
                onClick={() => onPageChange(entry)}
                className={`min-w-[30px] h-[30px] px-2 rounded-full text-xs font-extrabold transition cursor-pointer ${
                  entry === safePage
                    ? 'bg-nexa-brand text-white shadow-sm shadow-nexa-brand/25'
                    : 'border border-nexa-border bg-nexa-bg-base text-nexa-text-secondary hover:border-nexa-brand hover:text-nexa-brand'
                }`}
              >
                {entry}
              </button>
            ),
          )}
        </div>

        <button
          type="button"
          onClick={() => onPageChange(safePage + 1)}
          disabled={safePage >= totalPages}
          title="Next page"
          className="px-3 py-1.5 rounded-full border border-nexa-border bg-nexa-bg-base text-nexa-text-secondary hover:border-nexa-brand hover:text-nexa-brand disabled:opacity-40 disabled:pointer-events-none text-xs font-bold flex items-center gap-1 transition cursor-pointer"
        >
          <span className="hidden sm:inline">Next</span>
          <ChevronRight className="w-3.5 h-3.5" />
        </button>
        <button
          type="button"
          onClick={() => onPageChange(totalPages)}
          disabled={safePage >= totalPages}
          title="Last page"
          className="p-2 rounded-full border border-nexa-border bg-nexa-bg-base text-nexa-text-secondary hover:border-nexa-brand hover:text-nexa-brand disabled:opacity-40 disabled:pointer-events-none transition cursor-pointer"
        >
          <ChevronsRight className="w-3.5 h-3.5" />
        </button>
      </div>
    </div>
  );
};
