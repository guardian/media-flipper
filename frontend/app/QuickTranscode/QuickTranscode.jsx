import React from 'react';
import MenuBanner from "../MenuBanner.jsx";
import BasicUploadComponent from "./BasicUploadComponent.jsx";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import BytesFormatterImplementation from "../Common/BytesFormatterImplementation.jsx";
import css from "../inline-dialog.css";
import JobStatusComponent from "../JobList/JobStatusComponent.jsx";
import MediaFileInfo from "../JobList/MediaFileInfo.jsx";
import JobTemplateSelector from "./JobTemplateSelector.jsx";

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
            analysisResult: null,
            analysisTimer: null,
            jobTimer: null,
            jobId: null,
            fileName: null,
            settingsId: "63AD5DFB-F6F6-4C75-9F54-821D56458279",  //FAKE value for testing
            templateId: null
        };

        this.setStatePromise = this.setStatePromise.bind(this);
        this.newDataAvailable = this.newDataAvailable.bind(this);
        this.pollJobState = this.pollJobState.bind(this);
        this.pollAnalysisState = this.pollAnalysisState.bind(this);
    }

    setStatePromise(newState) {
        return new Promise((resolve, reject)=>this.setState(newState, ()=>resolve()));
    }

    /**
     * remove all timers in the state
     * @returns {Promise<unknown>}
     */
    clearAllTimers(){
        if(this.state.analysisTimer) window.clearInterval(this.state.analysisTimer);
        if(this.state.jobTimer) window.clearInterval(this.state.jobTimer);
        return this.setStatePromise({analysisTimer: null, jobTimer: null});
    }

    componentDidUpdate(prevProps, prevState, snapshot) {
        if(prevState.phase !== this.state.phase && this.state.phase===2){
            console.log("Entered analysis phase, starting timer");
            //const timerId = window.setInterval(this.pollAnalysisState, 500);
            //this.setStatePromise({analysisCompleted: false, analysisTimer: timerId});
            this.setStatePromise({analysisCompleted: false})
        }
        if(prevState.phase !== this.state.phase && this.state.phase!==2 && this.state.analysisTimer){
            console.log("Changed phase to not analysis with a timer set, removing it");
            window.clearTimeout(this.state.analysisTimer);
            this.setStatePromise({analysisTimer: null });
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
     * asks the server to create a new job.
     * @returns {Promise<string>} Resolves to the created job ID if successful or rejects otherwise
     */
    async createJobEntry() {
        //see webapp/jobs/jobrequest.go
        const requestContent = JSON.stringify({
            settingsId: this.state.settingsId,
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

    /**
     * called from a timer in phase 2 to check on analysis state
     * @returns {Promise<unknown>}
     */
    async pollAnalysisState() {
        const url = "/api/analysis/get?forId=" + this.state.jobId;
        const response = await fetch(url);
        if(response.status===200) {
            const content = await response.json();
            //FileFormatInfo, see models/fileformat.go
            const fileInfo = content.entry;
            return this.setStatePromise({analysisCompleted: true, analysisResult: fileInfo, phase: 3});
        } else if(response.status===500 || response.status===400){
            const content = await response.text();
            if(this.state.analysisTimer) window.clearInterval(this.state.analysisTimer);
            return this.setStatePromise({analysisCompleted: true, phase: 2, lastError: content, analysisTimer: null});
        } else {
            await response.body.cancel();
            console.log("Server returned ", response.status, response.statusText);
        }
    }

    async pollJobState() {
        const url = "/api/job/get?jobId=" + this.state.jobId;
        const response = await fetch(url);
        if(response.status===200) {
            const jobData = await response.json();
            console.log("updated job data: ", jobData.entry);
            return this.setStatePromise({jobStatus: {
                status: jobData.entry.status,
                    error: jobData.entry.error_message,
                completedSteps: jobData.entry.completed_steps,
                totalSteps: jobData.entry.steps.length
            }})
        } else {
            await this.clearAllTimers();
            return this.setStatePromise({lastError: "Could not get job: " + response.statusText})
        }
    }

    render() {
        return <div>
            <MenuBanner/>
            <div className="inline-dialog">
                <h2 className="inline-dialog-title">Quick transcode</h2>
                <div className="inline-dialog-content" style={{marginTop: "1em"}}>
                    <span className="transcode-info-block" style={{marginBottom: "1em", display:"block"}}>
                        <JobTemplateSelector value={this.state.templateId} onChange={evt=>this.setState({templateId: evt.target.value})}/>
                    </span>
                <BasicUploadComponent id="upload-box"
                                      loadStart={(file)=>this.setState({loading: true, fileName: file.name + " (" + BytesFormatterImplementation.getString(file.size) + " " + file.type + ")"})}
                                      loadCompleted={this.newDataAvailable}/>
                <label htmlFor="upload-box"><FontAwesomeIcon icon="upload" style={{marginRight: "4px"}}/>Upload a file</label>
                    {
                        this.state.analysisCompleted ? <span className="transcode-info-block"><MediaFileInfo jobId={this.state.jobId} fileInfo={this.state.analysisResult}/> </span>: ""
                    }

                <div id="placeholder" style={{height: "4em", display: "block", overflow: "hidden"}}>
                    <span className="transcode-info-block" style={{display: this.state.fileName ? "inherit" : "none"}}>{this.state.fileName}</span>

                    <span className="transcode-info-block" style={{display: this.state.phase<1 ? "none" : "block"}}>{
                        this.state.uploadCompleted ? "Uploading... Done!" : "Uploading..."
                    }</span>
                    <span className="transcode-info-block" style={{display: this.state.phase<2 ? "none" : "block"}}>
                        {
                            this.state.analysisCompleted ? "Analysing... Done!" : "Analysing..."
                        }
                    </span>
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