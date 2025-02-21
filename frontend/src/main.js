import './style.css';
import './app.css';

import { DateTime } from 'luxon';
import logo from './assets/images/romologo-monochrome.png';
//import {Login} from '../wailsjs/go/main/App';
import { initializeElectionPage} from './electionModule.js';
import { setupClock, updateClock, setTimeZone, populateTimeZones } from './clockModule.js';

// Set the logo
document.addEventListener('DOMContentLoaded', () => {
    document.querySelector('.main_logo').src = logo;
});

// Set up the login function
window.login = async function (email, password) {
    console.log("Attempting login with:", { email, password });
    if (email === "" || password === "") {
        alert("Please enter both email and password");
        return;
    }

    try {
        const response = await fetch('http://localhost:8080/login',
        {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ email, password }),
        });
        console.log("Raw response:", response);

        // ตรวจสอบ response type
        const contentType = response.headers.get("content-type");
        if (!contentType || !contentType.includes("application/json")) {
            throw new Error("Server did not return JSON");
        }

        const result = await response.json();

        if (response.ok) {
            // บันทึก email ของผู้ใช้ไว้ใน localStorage หรือ sessionStorage
            localStorage.setItem('userEmail', email);

            // check redirectURL recieve from backend
            if (result.redirectURL === '/survey') {
                showSurveyPage();
            } else if (result.redirectURL === '/election') {
                showElectionPage();
            } else {
                // กรณีอื่นๆ (ถ้ามี)
                console.error('Unexpected redirect URL:', result.redirectURL);
            }
        } else {
            // แสดงข้อความแจ้งเตือนกรณีล็อกอินไม่สำเร็จ
            alert(result.error || 'Login failed. Please try again.');
        }
    } catch (error) {
        console.error('Login error:', error);
        alert('An error occurred during login. Please try again.');
    }
};



function showElectionPage() {
    // Load the election-times-and-results.html content
    fetch('election-times-and-results.html')
        .then(response => response.text())
        .then(html => {
            document.getElementById('app').innerHTML = html;
            // Populate the election data  
             
            initializeElectionPage();  
            setupClock();
            setInterval(updateClock, 1000); 
            
            const timeZoneBtn = document.getElementById('changeTimeZoneBtn');
            const timeZoneDropdown = document.getElementById('timeZoneDropdown');

            timeZoneBtn.addEventListener('click', () => {
                if (timeZoneDropdown.style.display === 'none' || timeZoneDropdown.style.display === '') {
                    populateTimeZones();
                    timeZoneDropdown.style.display = 'block';
                    timeZoneDropdown.style.animation = 'dropdownUnfurl 0.4s ease-out';
                } else {
                    timeZoneDropdown.style.display = 'none';
                }
            });

            document.addEventListener('click', (event) => {
                const isClickInsideDropdown = timeZoneDropdown.contains(event.target);
                const isClickOnButton = timeZoneBtn.contains(event.target);
            
                if (!isClickInsideDropdown && !isClickOnButton) {
                    timeZoneDropdown.style.display = 'none';
                }
            });

        });
}




function showSurveyPage() {
    // Load the survey.html content
    fetch('survey.html')
        .then(response => response.text())
        .then(html => {
            document.getElementById('app').innerHTML = html;
            // Populate the survey data
            initializeSurveyPage();
        });
}

let selectedVote = null;
let navigationHistory = [];
let currentStep = -1;

function initializeSurveyPage() {
    showNext('voteContainer');

     // Initialize all event handlers
     document.querySelectorAll('[data-action]').forEach(element => {
        element.addEventListener('click', handleAction);
    });
}

function showNext(containerId) {
    document.querySelectorAll('.container').forEach(el => el.style.display = 'none');
    document.getElementById(containerId).style.display = 'block';

    // Update navigation history   
    if (navigationHistory[currentStep] !== containerId) {
        navigationHistory = navigationHistory.slice(0, currentStep + 1);
        navigationHistory.push(containerId);
        currentStep++;
    }
}    

function goBack() {
    if (currentStep > 0) {
        currentStep--;
        showNext(navigationHistory[currentStep]);
    }
}

function goForward() {
    if (currentStep == navigationHistory.length - 1) {                
        alert("Please submit your answer");
        return;
    }

    if (currentStep < navigationHistory.length - 1) {
        currentStep++;
        //console.log('Current Step after going forward:', currentStep);
        showNext(navigationHistory[currentStep]);
    }
}


function selectVote(value) {
    selectedVote = value;
    document.getElementById('voteMessage').textContent = "You selected: " + (value === 1 ? "+1" : value === 0 ? "±0" : "-1");
}

function submitVote() {
    if (selectedVote !== null) {
  
        if (selectedVote === 1) {
            showNext('nominationContainer');
        } else if (selectedVote === -1) {
            showNext('removalContainer');
        } else if (selectedVote === 0) {
            showNext('featureContainer');
        }
 
    } else {
        document.getElementById('voteMessage').textContent = "Please select an option before submitting your vote.";
    }
}


function submitNomination() {
    let nomineeName = document.getElementById("nomineeName").value;
    if (nomineeName) {

        document.getElementById("nominationMessage").textContent = "Nomination for " + nomineeName + " has been recorded. Results will be shown in 24 hours.";
        setTimeout(() => showNext('featureContainer'), 1000);
        
    } else {
        document.getElementById("nominationMessage").textContent = "Please enter a name before submitting your nomination.";
    }
}

function submitRemoval() {
    let removeName = document.getElementById("removeName").value;
    if (removeName) {
      
        document.getElementById("removalMessage").textContent = removeName + " has been proposed for removal. Results will be shown in 24 hours.";
        setTimeout(() => showNext('featureContainer'), 1000);
         
    } else {
        document.getElementById("removalMessage").textContent = "Please enter a name before submitting your removal proposal.";
    }
}

function submitFeature() {
    const feature = document.getElementById('featureInput').value;
    if (feature) {
      
        document.getElementById('responseMessage').textContent = "You proposed to add the software feature: " + feature + ". Results will be shown in 24 hours.";
        setTimeout(() => showNext('spendingContainer'), 1000);
            
          
    } else {
        document.getElementById('errorMessage').textContent = "Please enter a feature.";
    }
}

function submitProposal() {
    const amount = document.getElementById('amountInput').value;
    const purpose = document.getElementById('purposeInput').value;
    if (amount && purpose) {
       
        document.getElementById('responseMessage').textContent = "You proposed to spend $" + amount + " on " + purpose + ". Results will be shown in 24 hours.";
        setTimeout(() => showNext('questionContainer'), 1000);
           
    } else {
        document.getElementById('errorMessage').textContent = "Please enter both the amount and the purpose.";
    }
}

function submitQuestion() {
    const question = document.getElementById('questionInput').value;
    if (question.trim() !== "") {
       
        document.getElementById('responseMessage').textContent = `You asked: "${question}?"`;
        setTimeout(() => showNext('electionContainer'), 1000);
           
    } else {
        document.getElementById('responseMessage').textContent = "Please enter a valid question.";
    }
}

function submitElectionProposal() {
    const weeks = document.getElementById('weeksInput').value;
    if (weeks >= 1 && weeks <= 24) {
        
        document.getElementById('responseMessage').textContent = "You proposed the next election to be in " + weeks + " weeks. Results will be shown in 24 hours.";
        setTimeout(() => showNext('thresholdContainer'), 1000);
            
    } else {
        document.getElementById('errorMessage').textContent = "Please enter a valid number of weeks between 1 and 24.";
    }
}

function submitThreshold() {
    const numerator = parseInt(document.getElementById('numeratorInput').value);
    if (numerator) {
        
        document.getElementById('responseMessage').textContent = `You proposed that the number of votes needed for change is: \( \frac{${numerator}}{#business owners} \).`;
        submitSurvey();
           
    } else {
        document.getElementById('errorMessage').textContent = "Please enter a valid numerator.";
    }
}

async function submitSurvey() {
    const token = localStorage.getItem('jwtToken');
    if (!token) {
        alert('please login again');
        window.location.href = '/login';
        return;
    }

    const headers = {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
    };
    
    const surveyData = {
        response_data: {
            vote: {
                question_type: "vote",
                question_text: "I propose for the quantity of my country's business owners to:",
                response_value: selectedVote.toString()
            },
            nomination: {
                question_type: "nomination",
                question_text: document.getElementById("nomineeName") ? 
                    "I propose _____ to be a business owner starting from the next meeting:" :
                    "I propose _____ to no longer be a business owner starting from the next meeting:",
                response_value: document.getElementById("nomineeName")?.value || 
                            document.getElementById("removeName")?.value || ""
            },
            feature: {
                question_type: "feature",
                question_text: "I propose to add the software feature:",
                response_value: document.getElementById("featureInput")?.value || ""
            },
            spending: {
                question_type: "spending",
                question_text: "I propose that we spend:",
                response_value: `${document.getElementById("amountInput")?.value || ""} for ${document.getElementById("purposeInput")?.value || ""}`
            },
            question: {
                question_type: "question",
                question_text: "I want to ask if:",
                response_value: document.getElementById("questionInput")?.value || ""
            },
            election: {
                question_type: "election",
                question_text: "I propose that our next election will be in:",
                response_value: `${document.getElementById("weeksInput")?.value || ""} weeks`
            },
            threshold: {
                question_type: "threshold",
                question_text: "I propose that the number of votes needed for change is:",
                response_value: document.getElementById("numeratorInput")?.value || ""
            }
        }
    };

    if (!validateSurveyData(surveyData)) {
        alert('Please fill in all information');
        return;
    }

    try {
        const response = await fetch('http://localhost:8080/survey', {
            method: 'POST',
            headers: headers,
            body: JSON.stringify(surveyData)
        });

        if (response.status === 401) {
            handleUnauthorized();
            return;
        }
        
        const result = await response.json();
        handleSubmissionResult(result);
    } catch (error) {
        handleNetworkError(error);
    }
   
}

function validateSurveyData(data) {
    const rd = data.response_data;
    return (
      [-1, 0, 1].includes(Number(rd.vote.response_value)) &&
      rd.nomination.response_value.trim() !== "" &&
      rd.feature.response_value.trim() !== "" &&
      /^\d+\s+for\s+.+$/.test(rd.spending.response_value) &&
      rd.question.response_value.trim() !== "" &&
      /^\d+\s+weeks$/.test(rd.election.response_value) &&
      Number(rd.election.response_value.split(" ")[0]) >= 1 &&
      Number(rd.election.response_value.split(" ")[0]) <= 24 &&
      !isNaN(Number(rd.threshold.response_value))
    );
}
  

function handleSubmissionResult(result) {
    if (result.redirectURL) {
      window.location.href = result.redirectURL;
    }
    showSuccessMessage();
}
  
function handleUnauthorized() {
    localStorage.removeItem('jwtToken');
    window.location.href = '/login';
}

function handleNetworkError(error) {
    if (error.response?.status === 401) {
      localStorage.removeItem('jwtToken');
      window.location.href = '/login';
    } else {
      console.error('Network Error:', error);
      showErrorMessage('Connection failed. Please try again');
    }
}
  

function showSuccessMessage() {
    //document.getElementById('thankYouMessage').textContent = "Thank you! Your survey has been submitted.";
    showNext('thankYouContainer');

    // Add a delay before redirecting to the election page
    setTimeout(() => {
        showElectionPage();
    }, 3000); // 3 seconds delay
}


// Export functions for global access
window.selectVote = selectVote;
window.submitVote = submitVote;
window.goBack = goBack;
window.goForward = goForward;
window.submitNomination = submitNomination;
window.submitRemoval = submitRemoval;
window.submitFeature = submitFeature;
window.submitProposal = submitProposal;
window.submitQuestion = submitQuestion;
window.submitElectionProposal = submitElectionProposal;
window.submitThreshold = submitThreshold;


window.showElectionPage = showElectionPage;
window.setTimeZone = setTimeZone;

window.submitSurvey = submitSurvey;


