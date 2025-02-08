import { DateTime } from 'luxon';

let currentTimeZone = 'UTC';
let now;


let timeZones = [
    { name: 'Pacific/Kiritimati', city: 'Kiritimati, Kiribati' },
    { name: 'Pacific/Chatham', city: 'Chatham Islands, New Zealand' },
    { name: 'Pacific/Auckland', city: 'Auckland, New Zealand' },
    { name: 'Australia/Sydney', city: 'Sydney, Australia' },
    { name: 'Asia/Vladivostok', city: 'Vladivostok, Russia' },
    { name: 'Asia/Tokyo', city: 'Tokyo, Japan' },
    { name: 'Asia/Ulaanbaatar', city: 'Ulaanbaatar, Mongolia' },
    { name: 'Asia/Bangkok', city: 'Bangkok, Thailand' },
    { name: 'Asia/Yangon', city: 'Yangon, Myanmar' },
    { name: 'Asia/Dhaka', city: 'Dhaka, Bangladesh' },
    { name: 'Asia/Kathmandu', city: 'Kathmandu, Nepal' },
    { name: 'Asia/Kolkata', city: 'Kolkata, India' },
    { name: 'Asia/Karachi', city: 'Karachi, Pakistan' },
    { name: 'Asia/Dubai', city: 'Dubai, UAE' },
    { name: 'Asia/Tehran', city: 'Tehran, Iran' },
    { name: 'Europe/Moscow', city: 'Moscow, Russia' },
    { name: 'Europe/London', city: 'London, UK' },
    { name: 'UTC', city: 'UTC' },
    { name: 'Atlantic/Azores', city: 'Azores, Portugal' },
    { name: 'America/Noronha', city: 'Fernando de Noronha, Brazil' },
    { name: 'America/Argentina/Buenos_Aires', city: 'Buenos Aires, Argentina' },
    { name: 'America/New_York', city: 'New York, USA' },
    { name: 'America/Chicago', city: 'Chicago, USA' },
    { name: 'America/Denver', city: 'Denver, USA' },
    { name: 'America/Los_Angeles', city: 'Los Angeles, USA' },
    { name: 'America/Anchorage', city: 'Anchorage, USA' },
    { name: 'Pacific/Honolulu', city: 'Honolulu, USA' },
    { name: 'Pacific/Pago_Pago', city: 'Pago Pago, American Samoa' }
];

export function setupClock() {
  createTickMarks('tickMarksLeft');
  createTickMarks('tickMarksRight');
  drawStartUTCTime();
}

export function updateClock() {
    now = DateTime.now().setZone(currentTimeZone);
  const hours = now.hour;
  const minutes = now.minute;
  const seconds = now.second;

  const hourAngle = (hours % 12) * 30 + minutes * 0.5;
  const minuteAngle = minutes * 6 + seconds * 0.1;
  const secondAngle = seconds * 6;

  updateClockHand('hourHandLeft', hourAngle);
  updateClockHand('minuteHandLeft', minuteAngle);
  updateClockHand('secondHandLeft', secondAngle);
  updateClockHand('hourHandRight', hourAngle);
  updateClockHand('minuteHandRight', minuteAngle);
  updateClockHand('secondHandRight', secondAngle);
}

export function createTickMarks(elementId) {
  const tickMarks = document.getElementById(elementId);
  if (!tickMarks) return;

  for (let i = 0; i < 60; i++) {
    const angle = (i * 6) * Math.PI / 180;
    const x = 50 + 45 * Math.cos(angle);
    const y = 50 + 45 * Math.sin(angle);
    const circle = document.createElementNS('http://www.w3.org/2000/svg', 'circle');
    circle.setAttribute('cx', x);
    circle.setAttribute('cy', y);
    circle.setAttribute('r', i % 5 === 0 ? 1.5 : 1);
    circle.setAttribute('fill', 'black');
    tickMarks.appendChild(circle);
  }
}

export function updateClockHand(handId, angle) {
  document.getElementById(handId).setAttribute('transform', `rotate(${angle}, 50, 50)`);
}

export function drawStartUTCTime() {
  updateClock();
}

export function setTimeZone(newTimeZone) {
  currentTimeZone = newTimeZone;
  updateClock();
}



// Populate the dropdown menu with time zones and their current times
export function populateTimeZones() {
    // Clear the dropdown first
    timeZoneDropdown.innerHTML = '';

    // Sort time zones by current time in each zone
    timeZones = timeZones.map(zone => {
        const currentTime = DateTime.now().setZone(zone.name);
        return {
            ...zone,
            currentTime,
            formattedTime: currentTime.toLocaleString(DateTime.DATETIME_MED)
        };
    }).sort((a, b) => a.currentTime - b.currentTime);

    // Populate the dropdown
    timeZones.forEach(zone => {
        const option = document.createElement('div');
        option.value = zone.name;
        option.textContent = `${zone.city} - ${zone.formattedTime}`;
        option.style.cursor = 'pointer';
        option.style.padding = '1px';
        option.style.whiteSpace = 'nowrap';
        option.style.borderBottom = '1px solid #ccc';
        option.addEventListener('click', () => {
            currentTimeZone = zone.name;
            timeZoneDropdown.style.display = 'none';
            updateClock();
            console.log('timeZones:', currentTimeZone);
        });
        timeZoneDropdown.appendChild(option);
    });
}

//window.populateTimeZones = populateTimeZones;