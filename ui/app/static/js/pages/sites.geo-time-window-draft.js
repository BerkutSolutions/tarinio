export function readGeoTimeWindowDraftRows(container) {
  return Array.from(container.querySelectorAll("[data-geo-tw-countries]")).map((countriesInput) => {
    const index = String(countriesInput.dataset.geoTwCountries || "");
    const days = Array.from(container.querySelectorAll(`[data-geo-tw-day^="${index}-"]:checked`))
      .map((input) => Number.parseInt(String(input.dataset.geoTwDay || "").split("-").pop() || "-1", 10))
      .filter((day) => Number.isInteger(day) && day >= 0 && day <= 6);
    return {
      countries: String(countriesInput.value || "").split(",").map((item) => item.trim().toUpperCase()).filter(Boolean),
      action: String(container.querySelector(`[data-geo-tw-action="${index}"]`)?.value || "block"),
      days_of_week: days,
      hours_start: Number(container.querySelector(`[data-geo-tw-hours-start="${index}"]`)?.value || 0),
      hours_end: Number(container.querySelector(`[data-geo-tw-hours-end="${index}"]`)?.value || 0),
    };
  });
}
