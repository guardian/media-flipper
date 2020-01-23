import React from 'react';
import MenuBanner from "../MenuBanner.jsx";
import BasicUploadComponent from "./BasicUploadComponent.jsx";

class QuickTranscode extends React.Component {
    constructor(props) {
        super(props);

        this.state = {
            loading: false,
            lastError: null,
            jobId: null,
            settingsId: "63AD5DFB-F6F6-4C75-9F54-821D56458279"  //FAKE value for testing
        };

        this.setStatePromise = this.setStatePromise.bind(this);
        this.newDataAvailable = this.newDataAvailable.bind(this);

    }

    setStatePromise(newState) {
        return new Promise((resolve, reject)=>this.setState(newState, ()=>resolve()).catch(err=>reject(err)));
    }

    /**
     * asks the server to create a new job.
     * @returns {Promise<string>} Resolves to the created job ID if successful or rejects otherwise
     */
    async createJobEntry() {
        //see webapp/jobs/jobrequest.go
        const requestContent = JSON.stringify({
            settingsId: this.state.settingsId
        });

        const response = await fetch("/api/job/new",{method:"POST",body: requestContent,headers:{"Content-Type":"application/json"}});
        const responseBody = await response.text();

        if(response.status<200 || response.status>299){
            console.error("Server responded", response.status, response.statusText);
            console.error(responseBody);
            throw "Server error: " + response.statusText;
        }

        const responseJson = JSON.parse(responseBody);
        return responseJson.jobId;
    }

    /**
     * uploads a data buffer to the given job id
     * @param jobId
     * @param data
     * @returns {Promise<void>}
     */
    async uploadData(jobId, data) {
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
        await this.uploadData(newJobId, data);
    }

    newDataAvailable(data) {
        this.uploadProcess(data)
            .then(()=>this.setState({loading: false}))
            .catch(err=>this.setState({loading: false, lastError: err}))
    }

    render() {
        return <div>
            <MenuBanner/>
            <div className="centered">
                <label htmlFor="upload-box">Add your file here:</label>
                <BasicUploadComponent id="upload-box"
                                      loadStart={()=>this.setState({loading: true})}
                                      loadCompleted={this.newDataAvailable}/>
                <span className="loading" style={{display: this.state.loading ? "block" : "none"}}>Loading...</span>
                <span className="error" style={{display: this.state.lastError ? "block" : "none"}}>{this.state.lastError}</span>
            </div>
        </div>
    }
}

export default QuickTranscode;