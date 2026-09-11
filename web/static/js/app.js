document.addEventListener('DOMContentLoaded', () => {
  // Tab switching
  const tabShorten = document.getElementById('tab-shorten');
  const tabAnalytics = document.getElementById('tab-analytics');
  const sectionShorten = document.getElementById('section-shorten');
  const sectionAnalytics = document.getElementById('section-analytics');

  tabShorten.addEventListener('click', () => {
    tabShorten.classList.add('active');
    tabAnalytics.classList.remove('active');
    sectionShorten.style.display = 'block';
    sectionAnalytics.style.display = 'none';
  });

  tabAnalytics.addEventListener('click', () => {
    tabAnalytics.classList.add('active');
    tabShorten.classList.remove('active');
    sectionShorten.style.display = 'none';
    sectionAnalytics.style.display = 'block';
  });

  // Shorten Form
  const shortenForm = document.getElementById('shorten-form');
  const shortenError = document.getElementById('shorten-error');
  const shortenResult = document.getElementById('shorten-result');
  const shortUrlLink = document.getElementById('short-url-link');
  const copyBtn = document.getElementById('copy-btn');
  const viewAnalyticsBtn = document.getElementById('view-analytics-btn');

  shortenForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    shortenError.classList.remove('active');
    shortenResult.classList.remove('active');

    const longUrl = document.getElementById('long-url').value.trim();
    const customAlias = document.getElementById('custom-alias').value.trim();
    const expiresIn = document.getElementById('expires-in').value.trim();

    const payload = { url: longUrl };
    if (customAlias) payload.custom_alias = customAlias;
    if (expiresIn) payload.expires_in_minutes = parseInt(expiresIn, 10);

    try {
      const res = await fetch('/api/v1/shorten', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });

      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Failed to shorten URL');
      }

      shortUrlLink.href = data.short_url;
      shortUrlLink.textContent = data.short_url;
      shortenResult.classList.add('active');

      viewAnalyticsBtn.onclick = () => {
        tabAnalytics.click();
        document.getElementById('analytics-code').value = data.short_code;
        loadAnalytics(data.short_code);
      };
    } catch (err) {
      shortenError.textContent = err.message;
      shortenError.classList.add('active');
    }
  });

  // Copy button
  copyBtn.addEventListener('click', async () => {
    try {
      await navigator.clipboard.writeText(shortUrlLink.href);
      const originalText = copyBtn.textContent;
      copyBtn.textContent = 'Copied!';
      setTimeout(() => copyBtn.textContent = originalText, 2000);
    } catch (err) {
      console.error('Failed to copy', err);
    }
  });

  // Analytics Search
  const analyticsForm = document.getElementById('analytics-form');
  const analyticsCodeInput = document.getElementById('analytics-code');
  const analyticsError = document.getElementById('analytics-error');
  const analyticsResult = document.getElementById('analytics-result');
  const refreshAnalyticsBtn = document.getElementById('refresh-analytics-btn');
  const testRedirectBtn = document.getElementById('test-redirect-btn');

  let currentCode = '';

  analyticsForm.addEventListener('submit', (e) => {
    e.preventDefault();
    const code = analyticsCodeInput.value.trim();
    if (code) {
      loadAnalytics(code);
    }
  });

  refreshAnalyticsBtn.addEventListener('click', () => {
    if (currentCode) {
      loadAnalytics(currentCode);
    }
  });

  async function loadAnalytics(code) {
    currentCode = code;
    analyticsError.classList.remove('active');
    analyticsResult.style.display = 'none';

    try {
      const res = await fetch(`/api/v1/analytics/${encodeURIComponent(code)}`);
      const data = await res.json();

      if (!res.ok) {
        throw new Error(data.error?.message || 'Short URL not found');
      }

      document.getElementById('stat-total-clicks').textContent = data.total_clicks;
      document.getElementById('stat-clicks-today').textContent = data.clicks_today;
      document.getElementById('stat-clicks-7days').textContent = data.clicks_last_7_days;
      
      const lastClicked = data.last_clicked_at 
        ? new Date(data.last_clicked_at).toLocaleString() 
        : 'Never';
      document.getElementById('stat-last-clicked').textContent = lastClicked;

      document.getElementById('meta-short-code').textContent = data.short_code;
      document.getElementById('meta-original-url').textContent = data.original_url;
      document.getElementById('meta-original-url').href = data.original_url;
      document.getElementById('meta-created-at').textContent = new Date(data.created_at).toLocaleString();

      const statusBadge = document.getElementById('meta-status-badge');
      if (data.is_expired) {
        statusBadge.textContent = 'Expired';
        statusBadge.className = 'badge badge-expired';
      } else {
        statusBadge.textContent = 'Active';
        statusBadge.className = 'badge badge-active';
      }

      testRedirectBtn.href = `/${data.short_code}`;
      analyticsResult.style.display = 'block';
    } catch (err) {
      analyticsError.textContent = err.message;
      analyticsError.classList.add('active');
    }
  }
});
