import { useEffect, useMemo, useState } from "react";
import type {
  Listing,
  ListingResponse,
  MarketSnapshot,
  PriceObservation,
  SavedAlert,
  SourceStatus,
} from "./types";

const pageSize = 20;
const number = (value: number) => new Intl.NumberFormat("fa-IR").format(value);
const asciiDigits = (value: string) =>
  value
    .replace(/[۰-۹]/g, (digit) => String(digit.charCodeAt(0) - 1776))
    .replace(/[٠-٩]/g, (digit) => String(digit.charCodeAt(0) - 1632));
const numeric = (value: string) =>
  Number(asciiDigits(value).replace(/[^\d]/g, ""));
const isDemo = (item: Listing) => item.source_name.startsWith("demo-");
const sourceLabel = (item: Listing) =>
  isDemo(item)
    ? `فروشنده نمونه ${item.source_name.endsWith("a") ? "الف" : "ب"}`
    : item.source_name;
const groupKey = (item: Listing) =>
  [
    item.make,
    item.model,
    item.trim,
    item.production_year,
    item.year_calendar,
  ].join("|");

type Group = {
  key: string;
  title: string;
  year: number;
  offers: Listing[];
  lowest: number;
};

export default function App() {
  const [query, setQuery] = useState("");
  const [city, setCity] = useState("تهران");
  const [make, setMake] = useState("");
  const [maxPrice, setMaxPrice] = useState("");
  const [maxMileage, setMaxMileage] = useState("");
  const [yearFrom, setYearFrom] = useState("");
  const [condition, setCondition] = useState("");
  const [page, setPage] = useState(1);
  const [items, setItems] = useState<Listing[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [expanded, setExpanded] = useState<string | null>(null);
  const [selected, setSelected] = useState<Listing[]>([]);
  const [sources, setSources] = useState<SourceStatus[]>([]);
  const [market, setMarket] = useState<MarketSnapshot | null>(null);
  const [marketError, setMarketError] = useState(false);
  const [savedAlertToken, setSavedAlertToken] = useState(
    () => localStorage.getItem("rahcar-alert-token") || "",
  );
  const [alertCount, setAlertCount] = useState<number | null>(null);
  const [alertMessage, setAlertMessage] = useState("");

  useEffect(() => {
    fetch("/api/v1/sources")
      .then((response) => response.json())
      .then((data: { items: SourceStatus[] }) => setSources(data.items ?? []))
      .catch(() => {});
  }, []);

  useEffect(() => {
    fetch("/api/v1/market")
      .then((response) => {
        if (!response.ok) throw new Error("market prices unavailable");
        return response.json() as Promise<MarketSnapshot>;
      })
      .then(setMarket)
      .catch(() => setMarketError(true));
  }, []);

  const saveAlert = async () => {
    setAlertMessage("");
    try {
      const response = await fetch("/api/v1/alerts", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          query,
          city,
          make,
          max_price_toman: numeric(maxPrice) * 1_000_000,
        }),
      });
      if (!response.ok) throw new Error("ذخیره هشدار موفق نبود.");
      const alert: SavedAlert = await response.json();
      localStorage.setItem("rahcar-alert-token", alert.token);
      setSavedAlertToken(alert.token);
      setAlertCount(null);
      setAlertMessage(
        "جستجو ذخیره شد. آگهی‌های تازه را از همین صفحه بررسی کن.",
      );
    } catch (error) {
      setAlertMessage(
        error instanceof Error ? error.message : "خطا در ذخیره هشدار",
      );
    }
  };

  const checkAlert = async () => {
    try {
      const response = await fetch("/api/v1/alerts", {
        headers: { "X-Alert-Token": savedAlertToken },
      });
      if (!response.ok) throw new Error("بررسی هشدار موفق نبود.");
      const data: { count: number } = await response.json();
      setAlertCount(data.count);
      setAlertMessage("تعداد آگهی‌های تازه مطابق جستجوی ذخیره‌شده:");
    } catch (error) {
      setAlertMessage(
        error instanceof Error ? error.message : "خطا در بررسی هشدار",
      );
    }
  };

  useEffect(() => {
    setPage(1);
  }, [query, city, make, maxPrice, maxMileage, yearFrom, condition]);

  useEffect(() => {
    const controller = new AbortController();
    const timer = window.setTimeout(async () => {
      setLoading(true);
      setError("");
      const params = new URLSearchParams({
        limit: String(pageSize),
        page: String(page),
      });
      if (query.trim()) params.set("q", query.trim());
      if (city.trim()) params.set("city", city.trim());
      if (make.trim()) params.set("make", make.trim());
      if (numeric(maxPrice))
        params.set("max_price_toman", String(numeric(maxPrice) * 1_000_000));
      if (numeric(maxMileage))
        params.set("max_mileage_km", String(numeric(maxMileage) * 1_000));
      if (numeric(yearFrom)) params.set("year_from", String(numeric(yearFrom)));
      if (condition) params.set("condition", condition);
      try {
        const response = await fetch(`/api/v1/listings?${params}`, {
          signal: controller.signal,
        });
        if (!response.ok) throw new Error("دریافت آگهی‌ها موفق نبود.");
        const data: ListingResponse = await response.json();
        setItems(data.items ?? []);
        setTotal(data.count ?? 0);
      } catch (cause) {
        if (!controller.signal.aborted)
          setError(
            cause instanceof Error ? cause.message : "خطا در دریافت آگهی‌ها",
          );
      } finally {
        if (!controller.signal.aborted) setLoading(false);
      }
    }, 250);
    return () => {
      controller.abort();
      window.clearTimeout(timer);
    };
  }, [query, city, make, maxPrice, maxMileage, yearFrom, condition, page]);

  const groups = useMemo(() => {
    const grouped = new Map<string, Group>();
    for (const item of items) {
      const key = groupKey(item);
      const existing = grouped.get(key);
      if (existing) {
        existing.offers.push(item);
        existing.lowest = Math.min(existing.lowest, item.price_toman);
      } else {
        grouped.set(key, {
          key,
          title: `${item.make} ${item.model} ${item.trim}`.trim(),
          year: item.production_year,
          offers: [item],
          lowest: item.price_toman,
        });
      }
    }
    return [...grouped.values()]
      .map((group) => ({
        ...group,
        offers: group.offers.sort((a, b) => a.price_toman - b.price_toman),
      }))
      .sort((a, b) => a.lowest - b.lowest);
  }, [items]);

  const toggleSelected = (item: Listing) =>
    setSelected((current) =>
      current.some((entry) => entry.id === item.id)
        ? current.filter((entry) => entry.id !== item.id)
        : current.length < 3
          ? [...current, item]
          : current,
    );
  const clearFilters = () => {
    setQuery("");
    setCity("تهران");
    setMake("");
    setMaxPrice("");
    setMaxMileage("");
    setYearFrom("");
    setCondition("");
  };

  return (
    <>
      <header className="topbar">
        <div className="container topbar-inner">
          <a className="logo" href="/">
            <span className="logo-icon">ر</span>
            <span>راهکار</span>
          </a>
          <span className="topbar-subtitle">جستجو و مقایسه آگهی خودرو</span>
          <span className="topbar-chip">نسخه اولیه</span>
        </div>
      </header>

      <section className="hero">
        <div className="container hero-inner">
          <div className="hero-copy">
            <span className="eyebrow">خودروی مناسب، انتخاب آگاهانه</span>
            <h1>
              چند آگهی،
              <br />
              <em>یک نگاه روشن‌تر.</em>
            </h1>
            <p>
              قیمت، کارکرد و وضعیت خودروهای مشابه را کنار هم ببین و آگهی
              مناسب‌تر را پیدا کن.
            </p>
            <div className="hero-search">
              <span aria-hidden="true">⌕</span>
              <input
                aria-label="جستجوی خودرو"
                placeholder="نام خودرو را جستجو کن؛ مثلاً پژو ۲۰۷"
                value={query}
                onChange={(event) => setQuery(event.target.value)}
              />
            </div>
          </div>
          <div className="hero-art" aria-hidden="true">
            <div className="hero-circle">
              <span>🚘</span>
            </div>
            <div className="floating-card">
              <span>مقایسه آسان</span>
              <strong>قیمت · کارکرد · وضعیت</strong>
            </div>
          </div>
        </div>
      </section>

      <main className="container main-area">
        {items.some(isDemo) && (
          <div className="demo-banner">
            <span>ⓘ</span>
            <p>
              فیدهای نمونه این پروژه ساختگی هستند. قیمت‌ها و فروشنده‌های نمونه،
              اطلاعات واقعی بازار نیستند.
            </p>
          </div>
        )}
        <section className="market-section" aria-label="قیمت‌های بازار خودرو">
          <div className="market-heading">
            <div>
              <span className="section-label">داده واقعی بازار ایران</span>
              <h2>برآورد قیمت خودروهای کارکرده</h2>
              <p>قیمت میانی در سراسر ایران برای خودروی کارکرده با بدنه سالم؛ این عددها آگهی فروش یا قیمت معامله نیستند.</p>
            </div>
          </div>
          {market ? (
            <div className="market-grid">
              {market.items.map((item) => (
                <article className="market-card" key={item.source_url}>
                  <h3>{item.name} · مدل {number(item.model_year)}</h3>
                  <strong>{number(item.price_toman)} تومان</strong>
                  <p>{number(item.data_points)} قیمت اعلام‌شده · کارکرد معمول {number(item.mileage_km)} کیلومتر</p>
                  <a href={item.source_url} target="_blank" rel="noopener noreferrer">
                    نمودا · {new Date(market.computed_at).toLocaleDateString("fa-IR")}
                  </a>
                </article>
              ))}
            </div>
          ) : (
            <p className="market-status">{marketError ? "قیمت‌های بازار در دسترس نیستند." : "در حال دریافت قیمت‌های بازار..."}</p>
          )}
        </section>
        <div className="operations-row">
          <div className="source-summary">
            {sources.length === 0
              ? "هنوز وضعیت منبعی ثبت نشده است."
              : sources.map((source) => (
                  <span
                    key={source.source_name}
                    className={
                      source.status === "success" ? "source-good" : "source-bad"
                    }
                  >
                    {source.source_name}:{" "}
                    {source.status === "success"
                      ? "به‌روزرسانی موفق"
                      : "خطای به‌روزرسانی"}
                  </span>
                ))}
          </div>
          <div className="alert-actions">
            <button type="button" onClick={saveAlert}>
              ذخیره هشدار جستجو
            </button>
            {savedAlertToken && (
              <button type="button" onClick={checkAlert}>
                بررسی هشدار
              </button>
            )}
          </div>
        </div>
        {alertMessage && (
          <div className="alert-message" role="status">
            {alertMessage} {alertCount !== null && number(alertCount)}
          </div>
        )}
        <div className="heading-row">
          <div>
            <span className="section-label">بازار خودرو</span>
            <h2>آگهی‌های خودرو</h2>
            <p>آگهی‌های مشابه بر اساس مدل، تیپ و سال کنار هم قرار گرفته‌اند.</p>
          </div>
          <div className="result-count">
            {loading
              ? "در حال بارگذاری"
              : `${number(total)} آگهی · صفحه ${number(page)}`}
          </div>
        </div>

        <div className="layout">
          <aside className="filters">
            <div className="filter-head">
              <h3>فیلترها</h3>
              <button type="button" onClick={clearFilters}>
                پاک کردن
              </button>
            </div>
            <label>
              شهر
              <input
                value={city}
                onChange={(event) => setCity(event.target.value)}
                placeholder="مثلاً تهران"
              />
            </label>
            <label>
              برند
              <input
                value={make}
                onChange={(event) => setMake(event.target.value)}
                placeholder="مثلاً پژو"
              />
            </label>
            <label>
              سقف قیمت <small>میلیون تومان</small>
              <input
                inputMode="numeric"
                value={maxPrice}
                onChange={(event) => setMaxPrice(event.target.value)}
                placeholder="مثلاً ۹۰۰"
              />
            </label>
            <label>
              حداکثر کارکرد <small>هزار کیلومتر</small>
              <input
                inputMode="numeric"
                value={maxMileage}
                onChange={(event) => setMaxMileage(event.target.value)}
                placeholder="مثلاً ۵۰"
              />
            </label>
            <label>
              سال ساخت از <small>شمسی یا میلادی</small>
              <input
                inputMode="numeric"
                value={yearFrom}
                onChange={(event) => setYearFrom(event.target.value)}
                placeholder="مثلاً ۱۴۰۰"
              />
            </label>
            <label>
              وضعیت بدنه
              <select
                value={condition}
                onChange={(event) => setCondition(event.target.value)}
              >
                <option value="">همه وضعیت‌ها</option>
                <option>بدون رنگ</option>
                <option>یک لکه رنگ</option>
                <option>رنگ‌شدگی</option>
              </select>
            </label>
            <div className="filter-note">
              سال و تیپ خودرو در هر گروه یکسان است. قیمت پایین‌تر همیشه به معنی
              معامله بهتر نیست.
            </div>
          </aside>

          <div className="results">
            {error ? (
              <div className="state-card">
                <h3>اتصال به آگهی‌ها برقرار نشد</h3>
                <p>{error} مطمئن شو سرویس Go و پایگاه داده اجرا شده‌اند.</p>
              </div>
            ) : loading ? (
              <div className="state-card">
                <p>در حال دریافت آگهی‌ها...</p>
              </div>
            ) : groups.length === 0 ? (
              <div className="state-card">
                <h3>آگهی‌ای پیدا نشد</h3>
                <p>فیلترها را تغییر بده یا فیدهای نمونه را وارد کن.</p>
              </div>
            ) : (
              groups.map((group) => (
                <article className="group-card" key={group.key}>
                  <div className="group-summary">
                    <div className="car-thumb" aria-hidden="true">
                      🚘
                    </div>
                    <div className="group-title">
                      <span>مدل {number(group.year)}</span>
                      <h3>{group.title}</h3>
                      <p>
                        {number(group.offers.length)} آگهی از{" "}
                        {number(
                          new Set(group.offers.map((item) => item.source_name))
                            .size,
                        )}{" "}
                        منبع در این صفحه
                      </p>
                    </div>
                    <div className="group-action">
                      <small>شروع قیمت درخواستی از</small>
                      <strong>
                        {number(group.lowest)} <span>تومان</span>
                      </strong>
                      <button
                        type="button"
                        aria-expanded={expanded === group.key}
                        onClick={() =>
                          setExpanded(expanded === group.key ? null : group.key)
                        }
                      >
                        {expanded === group.key
                          ? "بستن آگهی‌ها"
                          : "دیدن آگهی‌ها"}{" "}
                        <span>←</span>
                      </button>
                    </div>
                  </div>
                  {expanded === group.key && (
                    <div className="offer-list">
                      {group.offers.map((item) => (
                        <div className="offer" key={item.id}>
                          <div className="offer-source">
                            <b>{sourceLabel(item)}</b>
                            <small>
                              {item.city} ·{" "}
                              {isDemo(item)
                                ? "داده نمونه"
                                : new Date(
                                    item.last_seen_at,
                                  ).toLocaleDateString("fa-IR")}
                            </small>
                            {item.potential_duplicate_count > 0 && (
                              <small className="duplicate-warning">
                                احتمال ثبت تکراری در منبع دیگر
                              </small>
                            )}
                          </div>
                          <div>
                            <small>کارکرد</small>
                            <b>{number(item.mileage_km)} کیلومتر</b>
                          </div>
                          <div>
                            <small>وضعیت بدنه</small>
                            <b>{item.body_condition || "اعلام نشده"}</b>
                          </div>
                          <div className="offer-price">
                            <small>قیمت درخواستی</small>
                            <b>{number(item.price_toman)} تومان</b>
                          </div>
                          <div className="offer-actions">
                            <button
                              type="button"
                              className={
                                selected.some((entry) => entry.id === item.id)
                                  ? "compare selected"
                                  : "compare"
                              }
                              onClick={() => toggleSelected(item)}
                              aria-pressed={selected.some(
                                (entry) => entry.id === item.id,
                              )}
                            >
                              {selected.some((entry) => entry.id === item.id)
                                ? "انتخاب شد"
                                : "مقایسه"}
                            </button>
                            {!isDemo(item) && (
                              <a
                                href={item.source_url}
                                target="_blank"
                                rel="noopener noreferrer"
                              >
                                آگهی اصلی ↗
                              </a>
                            )}
                          </div>
                          <OfferHistory listingID={item.id} />
                        </div>
                      ))}
                    </div>
                  )}
                </article>
              ))
            )}
            {!loading && !error && total > pageSize && (
              <div className="pagination">
                <button
                  type="button"
                  disabled={page === 1}
                  onClick={() => setPage(page - 1)}
                >
                  صفحه قبل
                </button>
                <span>
                  صفحه {number(page)} از {number(Math.ceil(total / pageSize))}
                </span>
                <button
                  type="button"
                  disabled={page * pageSize >= total}
                  onClick={() => setPage(page + 1)}
                >
                  صفحه بعد
                </button>
              </div>
            )}
          </div>
        </div>

        {selected.length > 0 && (
          <section
            className="comparison"
            aria-label="مقایسه آگهی‌های انتخاب شده"
          >
            <div className="comparison-head">
              <div>
                <span className="section-label">مقایسه کنار هم</span>
                <h2>{number(selected.length)} آگهی انتخاب شده</h2>
              </div>
              <button type="button" onClick={() => setSelected([])}>
                پاک کردن انتخاب‌ها
              </button>
            </div>
            <div className="comparison-grid">
              {selected.map((item) => (
                <div className="comparison-card" key={item.id}>
                  <h3>
                    {item.make} {item.model} {item.trim}
                  </h3>
                  <p>{sourceLabel(item)}</p>
                  <dl>
                    <div>
                      <dt>سال</dt>
                      <dd>{number(item.production_year)}</dd>
                    </div>
                    <div>
                      <dt>قیمت</dt>
                      <dd>{number(item.price_toman)} تومان</dd>
                    </div>
                    <div>
                      <dt>کارکرد</dt>
                      <dd>{number(item.mileage_km)} کیلومتر</dd>
                    </div>
                    <div>
                      <dt>بدنه</dt>
                      <dd>{item.body_condition || "اعلام نشده"}</dd>
                    </div>
                    <div>
                      <dt>شهر</dt>
                      <dd>{item.city}</dd>
                    </div>
                  </dl>
                </div>
              ))}
            </div>
          </section>
        )}
      </main>
      <footer>
        <div className="container">
          راهکار · مقایسه آگهی خودرو از منابع دارای مجوز
        </div>
      </footer>
    </>
  );
}

function OfferHistory({ listingID }: { listingID: number }) {
  const [open, setOpen] = useState(false);
  const [points, setPoints] = useState<PriceObservation[] | null>(null);
  const [error, setError] = useState("");

  const toggle = async () => {
    if (open) {
      setOpen(false);
      return;
    }
    setOpen(true);
    if (points !== null) return;
    try {
      const response = await fetch(`/api/v1/listings/${listingID}/history`);
      if (!response.ok) throw new Error("تاریخچه قیمت دریافت نشد.");
      const data: { items: PriceObservation[] } = await response.json();
      setPoints(data.items ?? []);
    } catch (cause) {
      setError(
        cause instanceof Error ? cause.message : "خطا در دریافت تاریخچه",
      );
    }
  };

  return (
    <div className="history-panel">
      <button type="button" onClick={toggle} aria-expanded={open}>
        تاریخچه قیمت {open ? "⌃" : "⌄"}
      </button>
      {open &&
        (error ? (
          <p>{error}</p>
        ) : points === null ? (
          <p>در حال دریافت...</p>
        ) : points.length === 0 ? (
          <p>هنوز تاریخچه‌ای ثبت نشده است.</p>
        ) : (
          <ol>
            {points.map((point) => (
              <li key={point.observed_at}>
                <span>
                  {new Date(point.observed_at).toLocaleDateString("fa-IR")}
                </span>
                <strong>{number(point.price_toman)} تومان</strong>
              </li>
            ))}
          </ol>
        ))}
    </div>
  );
}
