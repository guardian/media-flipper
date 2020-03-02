import React from 'react';
import MenuBanner from "../MenuBanner.jsx";
import BasicUploadComponent from "./BasicUploadComponent.jsx";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import BytesFormatterImplementation from "../Common/BytesFormatterImplementation.jsx";
import css from "../inline-dialog.css";
import JobStatusComponent from "../JobList/JobStatusComponent.jsx";
import MediaFileInfo from "../JobList/MediaFileInfo.jsx";
import JobTemplateSelector from "./JobTemplateSelector.jsx";
import JobProgressComponent from "./JobProgressComponent.jsx";

class QuickTranscode extends React.Component {
    //job status values from models/jobentry.go
    JOB_PENDING = 0;
    JOB_STARTED = 1;
    JOB_COMPLETED = 2;
    JOB_FAILED = 3;

    constructor(props) {
        super(props);

        this.state = {
            phase: 0,
            uploadCompleted: false,
            jobStatus: {
                status: 0,
                completedSteps: 0,
                totalSteps: 0
            },
            lastError: null,
            analysisCompleted: false,
            analysisResultId: null,
            analysisResult: null,
            jobTimer: null,
            jobId: null,
            fileName: null,
            templateId: "",
            templateEntries: [],
            selectedJobSteps: []
        };

        this.setStatePromise = this.setStatePromise.bind(this);
        this.newDataAvailable = this.newDataAvailable.bind(this);
        this.pollJobState = this.pollJobState.bind(this);
    }

    setStatePromise(newState) {
        return new Promise((resolve, reject)=>this.setState(newState, ()=>resolve()));
    }

    /**
     * remove all timers in the state
     * @returns {Promise<unknown>}
     */
    clearAllTimers(){
        if(this.state.jobTimer) window.clearInterval(this.state.jobTimer);
        return this.setStatePromise({jobTimer: null});
    }

    async loadTemplatesList() {
        await this.setStatePromise({loading: true, lastError: null});

        const response = await fetch("/api/jobtemplate");
        if(response.status===200){
            const serverData = await response.json();
            const templateEntries = serverData.entries;
            const templateIdUpdate = templateEntries.length>0 ? {templateId: templateEntries[0].Id} : {};

            await this.setStatePromise(Object.assign({loading: false, lastError: null, templateEntries: templateEntries}, templateIdUpdate));
        } else {
            const bodyText = await response.text();

            return this.setStatePromise({loading: false, lastError: bodyText})
        }
    }

    componentDidMount() {
        this.loadTemplatesList();
    }

    componentWillUnmount() {
        if(this.state.jobTimer) {
            window.clearTimeout(this.state.jobTimer);
            this.setState({jobTimer: null});
        }
    }

    async componentDidUpdate(prevProps, prevState, snapshot) {
        if(prevState.phase !== this.state.phase && this.state.phase===2){
            await this.setStatePromise({analysisCompleted: false})
        }

        if(prevState.templateId!==this.state.templateId) {
            const matchingEntries = this.state.templateEntries.filter(ent=>ent.Id===this.state.templateId);
            if(matchingEntries.length>0){
                await this.setStatePromise({selectedJobSteps: matchingEntries[0].Steps})
            }
        }
        if(prevState.analysisResultId !== this.state.analysisResultId) {
            if(this.state.analysisResultId!=="00000000-0000-0000-0000-000000000000") {
                console.log("Analysis result changed");
                await this.loadAnalysisData();
            }
        }

        if(prevState.jobStatus !== this.state.jobStatus){
            console.log("Job status changed from ", prevState.jobStatus, " to ", this.state.jobStatus);
            switch(this.state.jobStatus.status){
                case this.JOB_COMPLETED:
                    this.clearAllTimers();
                    break;
                case this.JOB_FAILED:
                    this.clearAllTimers();
                    break;
                default:
                    break;
            }
        }
    }

    /**
     * loads the data associated with the analysis ID in the state
     * @returns {Promise<void>}
     */
    async loadAnalysisData() {
        const response = await fetch("/api/analysis/get?forId=" + this.state.analysisResultId);
        if(response.status===200){
            const parsedData = await response.json();
            if(!parsedData.hasOwnProperty("entry")){
                return this.setStatePromise({lastError: "invalid response from /api/analysis/get"});
            } else {
                return this.setStatePromise({analysisResult: parsedData.entry, analysisCompleted: true});
            }
        } else {
            const errorText = await response.text();
            return this.setStatePromise({lastError: errorText});
        }
    }

    /**
     * asks the server to create a new job.
     * @returns {Promise<string>} Resolves to the created job ID if successful or rejects otherwise
     */
    async createJobEntry() {
        //see webapp/jobs/jobrequest.go
        const requestContent = JSON.stringify({
            jobTemplateId: this.state.templateId,
        });

        const response = await fetch("/api/job/new",{method:"POST",body: requestContent,headers:{"Content-Type":"application/json"}});
        const responseBody = await response.text();

        if(response.status<200 || response.status>299){
            console.error("Server responded", response.status, response.statusText);
            console.error(responseBody);
            throw "Server error: " + response.statusText;
        }

        const responseJson = JSON.parse(responseBody);
        return responseJson.jobContainerId;
    }

    /**
     * uploads a data buffer to the given job id
     * @param jobId
     * @param data
     * @returns {Promise<void>}
     */
    async uploadData(jobId, data) {
        console.log("uploadData: ", data);
        const response = await fetch("/api/flip/upload?forJob="+jobId, {method:"POST", body: data, headers:{"Content-Type":"application/octet-stream"}});
        const responseBody = await response.text();

        if(response.status<200 || response.status>299){
            console.error("Server responded", response.status, response.statusText);
            console.error(responseBody);
            throw "Server error: " + response.statusText;
        }
    }

    async uploadProcess(data) {
        const newJobId = await this.createJobEntry();
        console.log("Job created with id ",newJobId);
        await this.setStatePromise({jobId: newJobId, phase: 1, uploadCompleted: false, jobTimer: window.setInterval(this.pollJobState, 500)});
        await this.uploadData(newJobId, data);
    }

    newDataAvailable(data) {
        this.uploadProcess(data)
            .then(()=>this.setState({phase: 2, uploadCompleted: true}))
            .catch(err=>this.setState({loading: false, lastError: err}))
    }

    getAnalysisResultId(stepList) {
        const analysisStep = this.findAnalysisStep(stepList);
        if(analysisStep==null) {
            return null;
        } else {
            return analysisStep.analysisResult;
        }
    }

    /**
     * called from a timer to check the job status at regular intervals
     * @returns {Promise<unknown>}
     */
    async pollJobState() {
        const url = "/api/job/get?jobId=" + this.state.jobId;
        const response = await fetch(url);
        if(response.status===200) {
            const jobData = await response.json();

            const analysisResultId = this.getAnalysisResultId(jobData.entry.steps);

            const analysisResultUpdate = analysisResultId===null ? {} : {analysisResultId: analysisResultId};

            console.log("analysisResultUpdate: ", analysisResultUpdate);

            return this.setStatePromise(Object.assign({}, analysisResultUpdate, {jobStatus: {
                status: jobData.entry.status,
                error: jobData.entry.error_message,
                completedSteps: jobData.entry.completed_steps,
                totalSteps: jobData.entry.steps.length
            }}))
        } else {
            await this.clearAllTimers();
            return this.setStatePromise({lastError: "Could not get job: " + response.statusText})
        }
    }

    findAnalysisStep(jobSteps) {
        const resultList = jobSteps.filter(s=>s.stepType==="analysis");
        return resultList.length>0 ? resultList[0] : null;
    }

    render() {
        return <div>
            <MenuBanner/>
            <div className="inline-dialog">
                <h2 className="inline-dialog-title">Quick transcode</h2>
                <div className="inline-dialog-content" style={{marginTop: "1em"}}>
                    <span className="transcode-info-block" style={{marginBottom: "1em", display:"block"}}>
                        <JobTemplateSelector value={this.state.templateId}
                                             onChange={evt=>this.setState({templateId: evt.target.value})}
                                             jobTemplateList={this.state.templateEntries}
                        />
                    </span>
                <BasicUploadComponent id="upload-box"
                                      loadStart={(file)=>this.setState({loading: true, fileName: file.name + " (" + BytesFormatterImplementation.getString(file.size) + " " + file.type + ")"})}
                                      loadCompleted={this.newDataAvailable}/>
                <label htmlFor="upload-box" style={{display: this.state.phase<1 ? "inherit" : "none"}}><FontAwesomeIcon icon="upload" style={{marginRight: "4px"}}/>Upload a file</label>
                    {
                        this.state.analysisCompleted ? <span className="transcode-info-block"><MediaFileInfo jobId={this.state.jobId} fileInfo={this.state.analysisResult}/> </span>: ""
                    }

                <div id="placeholder" style={{height: "6em", display: "block", overflow: "hidden"}}>
                    <span className="transcode-info-block" style={{display: this.state.fileName ? "inherit" : "none"}}>{this.state.fileName}</span>

                    <span className="transcode-info-block" style={{display: this.state.phase<1 ? "none" : "block"}}>{
                        this.state.uploadCompleted ? "Uploading... Done!" : "Uploading..."
                    }</span>
                    <JobProgressComponent jobStepList={this.state.selectedJobSteps}
                                          currentJobStep={this.state.jobStatus.completedSteps}
                                          className=""
                                          hidden={this.state.phase<2}
                    />
                    <span className="error-text" style={{display: this.state.lastError ? "block" : "none"}}>{this.state.lastError}</span>
                </div>

                    <span className="transcode-info-block" style={{fontWeight: "bold"}}>Job is <JobStatusComponent status={this.state.jobStatus.status}/>, completed {this.state.jobStatus.completedSteps} out of {this.state.jobStatus.totalSteps} steps</span>
                    <span className="transcode-info-block error-text" style={{display: this.state.jobStatus.error ? "inherit" : "none"}}>Failed: {this.state.jobStatus.error}</span>
                </div>
            </div>
        </div>
    }
}

export default QuickTranscode;