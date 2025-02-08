import { setupClock, updateClock } from './clockModule.js';
let currentMonth = 1;
let currentYear = 2025;

export function initializeElectionPage() {
  renderPollingResults();
  updateCalendarHeader();
  updateCalendarDays();
}

export function renderPollingResults() {
  const pollingResultsContainer = document.getElementById('pollingResults');
  if (!pollingResultsContainer) {
    console.error('Polling results container not found');
    return;
  }
  
  pollingData.forEach(data => {
    const percentage = generateRandomPercentage();
    const resultItem = `
      <div class="result-item">
        <div class="question">${data.question}</div>
        <div class="bar-container">
          <div class="bar" style="width: ${percentage}%; background-color: black;">${percentage}%</div>
        </div>
      </div>
    `;
    pollingResultsContainer.insertAdjacentHTML('beforeend', resultItem);
  });
}

export function updateCalendarHeader() {
  const monthNames = ["January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"];
  document.getElementById("calendarHeader").innerText = `${monthNames[currentMonth]} ${currentYear}`;
}

export function updateCalendarDays() {
    const calendarBody = document.getElementById("calendarBody");
    if (!calendarBody) return;

    calendarBody.innerHTML = "";
    const firstDay = new Date(currentYear, currentMonth, 1).getDay();
    const daysInMonth = new Date(currentYear, currentMonth + 1, 0).getDate();
    const daysInPrevMonth = new Date(currentYear, currentMonth, 0).getDate();

    let dayCounter = 1;
    const totalCells = 42; // 6 rows * 7 days

    for (let i = 0; i < 6; i++) {
        const row = document.createElement("tr");
        
        for (let j = 0; j < 7; j++) {
        const cell = document.createElement("td");
        let cellDate;

        if (i === 0 && j < firstDay) {
            // Previous month
            cellDate = daysInPrevMonth - firstDay + j + 1;
            cell.className = "GrayCalendarCell";
        } else if (dayCounter > daysInMonth) {
            // Next month
            cellDate = dayCounter - daysInMonth;
            cell.className = "GrayCalendarCell";
            dayCounter++;
        } else {
            // Current month
            cellDate = dayCounter;
            cell.className = "CalendarCell";
            if (new Date(currentYear, currentMonth, cellDate).getDay() === 4) {
            cell.style.textDecoration = "underline";
            }
            dayCounter++;
        }

        cell.textContent = cellDate;
        row.appendChild(cell);
        }

        calendarBody.appendChild(row);
        if (dayCounter > daysInMonth && i >= 4) break; // Stop if we've filled all days and completed at least 5 rows
    }
}
  
export function nextMonth() {
    currentMonth++;
    if (currentMonth > 11) {
        currentMonth = 0;
        currentYear++;
    }
    updateCalendarHeader();
    updateCalendarDays();
}

export function previousMonth() {
    currentMonth--;
    if (currentMonth < 0) {
        currentMonth = 11;
        currentYear--;
    }
    updateCalendarHeader();
    updateCalendarDays();
}


const pollingData = [
    {
        question: "I nominate ________ to become a business owner."
    },
    {
        question: "I propose ________ to no longer be a business owner starting from the next meeting."
    },
    {
        question: "I propose that our next election will be every ____ weeks for the 24-hour period starting at UTC 00:00:00."
    },
    {
        question: "I propose that we spend $_______ on _______."
    },
    {
        question: "I propose to add the software feature: ________."
    },
    {
        question: "I propose that the number of votes needed for change is greater than or equal to ____."
    },
    {
        question: "I want to ask if: _____?"
    }
];

function generateRandomPercentage() {
  return Math.floor(Math.random() * 101);
}

window.previousMonth = previousMonth;
window.nextMonth = nextMonth;
