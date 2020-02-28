import React from 'react';
import PropTypes from 'prop-types';
import css from "./JobList.css";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import HidableExpander from "../Common/HidableExpander.jsx";
import JobStatusComponent from "./JobStatusComponent.jsx";
import MenuBanner from "../MenuBanner.jsx";
import MediaFileInfo from "./MediaFileInfo.jsx";
import TimestampFormatter from "../Common/TimestampFormatter.jsx";
import ThumbnailPreview from "./ThumbnailPreview.jsx";
import Modal from 'react-responsive-modal';
import MediaPreview from "./MediaPreview.jsx";
import JobStatusSummary from "./JobStatusSummary.jsx";

class JobList extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            jobList: [],
            loading: false,
            lastError: null,
            jobTemplateLookup: {},
            nextPageCursor: 0,
            showModal: false,
            modalThumbnailId: null,
            showLogsFor: null,
            currentLogContent: null,
            currentStatusFilter: "",
        };

        this.setStatusFilter = this.setStatusFilter.bind(this);
    }

    refreshData() {
        this.setState({loading: true}, async ()=>{
            await this.loadJobTemplateLookup();
            await this.loadJobListPage();
        });
    }

    componentDidMount() {
        this.refreshData();
    }

    componentDidUpdate(prevProps, prevState, snapshot) {
        if(prevState.showLogsFor!==this.state.showLogsFor && this.state.showLogsFor!==null) {
          this.loadInLogsContent();
        }
    }

    async loadInLogsContent() {
        const response = await fetch("/api/job/logs?stepId=" + this.state.showLogsFor);
        const content = await response.text();
        if(response.status===200){
            this.setState({currentLogContent: content})
        } else {
            const errMsg = "Could not load logs, server responded " + response.status + " " + response.statusText;
            this.setState({currentLogContent: errMsg + "\n" + content});
        }
    }

    async loadJobTemplateLookup() {
        const response = await fetch("/api/jobtemplate");
        if(response.status===200){
            const data = await response.json();
            const newLookup = data.entries.reduce((acc,ent)=>{
                let newEntry = {};
                newEntry[ent.Id] = ent.JobTypeName;
                return Object.assign(acc, newEntry);
            }, {});
            return new Promise((resolve,reject)=>this.setState({jobTemplateLookup: newLookup}, ()=>resolve()));
        } else {
            const errorText = await response.text();
            return new Promise((resolve, reject)=>this.setState({lastError: errorText}, ()=>resolve()));
        }
    }

    getQueryParams() {
        if(this.props.location.search.length<2) {
            return {}
        }
        const baseString = this.props.location.search.slice(1);  //slice off the leading ?
        const parts = baseString.split("&");                        //split on & character
        const result = parts.reduce((acc,elem)=>{                   //split each entry on = and add it to an object
            console.log("elem is ", elem);
            const kv=elem.split("=");
            const newentry = {};
            console.log("kv is ", kv);
            newentry[kv[0]] = kv[1];
            return Object.assign(newentry, acc)
        },{});
        return result;
    }

    async loadJobListPage() {
        const qps = this.getQueryParams();
        console.log("got queryparams: ", qps);
        let url;
        if(qps.hasOwnProperty("jobId")) {
            url = "/api/job?jobId=" + qps["jobId"];
        } else {
            url = "/api/job"
        }

        if(qps.length===0){
            url = "/api/job"
        } else {
            url = url + this.props.location.search
        }

        const response = await fetch(url);
        if(response.status===200){
            const data = await response.json();
            return new Promise((resolve, reject)=>this.setState(prevState=>{
                return {loading: false,
                    jobList: prevState.jobList.concat(data.entries),
                    nextPageCursor: data.nextCursor}
            }, ()=>resolve()));
        } else {
            const text = await response.text();
            return new Promise((resolve, reject)=>this.setState({lastError: text, loading: false}, ()=>resolve()));
        }
    }

    getTemplateName(templateId){
        if(this.state.jobTemplateLookup.hasOwnProperty(templateId)){
            return this.state.jobTemplateLookup[templateId];
        } else {
            return "unknown template Id" + templateId;
        }
    }

    renderJobStepDetails(step, idx){
        switch(step.stepType){
            case "analysis":
                return <div className="job-list-container">
                    <div className="job-list-entry-cell baseline"><FontAwesomeIcon icon="wrench"/>  Step {idx+1}</div>
                    <div className="job-list-entry-cell baseline">
                        <JobStatusComponent status={step.jobStepStatus}/><br/>
                        <a href="#" onClick={evt=>{ evt.preventDefault(); this.setState({showLogsFor: step.id})}}>Show logs...</a>
                    </div>
                    <div className="job-list-entry-cell baseline">Format Analysis</div>
                    <div className="job-list-entry-cell baseline">{step.analysisResult && step.analysisResult!=="00000000-0000-0000-0000-000000000000" ? <MediaFileInfo jobId={step.analysisResult} initialExpanderState={true}/> : <span/>}</div>
                    <div className="job-list-entry-cell wide">{step.errorMessage}</div>
                    </div>;
            case "thumbnail":
                /*pad out with empty divs so that the columns align*/
                return <div className="job-list-container">
                    <div className="job-list-entry-cell baseline"><FontAwesomeIcon icon="wrench"/>  Step {idx+1}</div>
                    <div className="job-list-entry-cell baseline">
                        <JobStatusComponent status={step.jobStepStatus}/><br/>
                        <a href="#" onClick={evt=>{ evt.preventDefault(); this.setState({showLogsFor: step.id})}}>Show logs...</a>
                    </div>
                    <div className="job-list-entry-cell baseline">Generate thumbnail</div>
                    <div className="job-list-entry-cell baseline"><ThumbnailPreview fileId={step.thumbnailResult} clickable={true} className="thumbnail-preview" onClick={()=>this.setState({showModal: true, modalThumbnailId: step.thumbnailResult})}/></div>
                    <div className="job-list-entry-cell wide">{step.errorMessage}</div>
                </div>;
            case "transcode":
                return <div className="job-list-container">
                    <div className="job-list-entry-cell baseline"><FontAwesomeIcon icon="wrench"/>  Step {idx+1}</div>
                    <div className="job-list-entry-cell baseline">
                        <JobStatusComponent status={step.jobStepStatus}/><br/>
                        <a href="#" onClick={evt=>{ evt.preventDefault(); this.setState({showLogsFor: step.id})}}>Show logs...</a>
                    </div>
                    <div className="job-list-entry-cell baseline">Transcode</div>
                    <div className="job-list-entry-cell baseline"><MediaPreview className="thumbnail-preview" fileId={step.transcodeResult}/></div>
                    <div className="job-list-entry-cell wide">{step.errorMessage}</div>
                </div>;
            case "custom":
                return <div className="job-list-container">
                    <div className="job-list-entry-cell baseline"><FontAwesomeIcon icon="wrench"/>  Step {idx+1}</div>
                    <div className="job-list-entry-cell baseline">
                        <JobStatusComponent status={step.jobStepStatus}/><br/>
                        <a href="#" onClick={evt=>{ evt.preventDefault(); this.setState({showLogsFor: step.id})}}>Show logs...</a>
                    </div>
                    <div className="job-list-entry-cell baseline">Custom ({step.templateFile})</div>
                    <div className="job-list-entry-cell baseline"/>
                    <div className="job-list-entry-cell wide">{step.errorMessage}</div>
                </div>;
            default:
                return <div className="job-list-container"><div className="job-list-entry-cell wide">Unknown job step type {step.stepType}</div></div>
        }
    }

    setStatusFilter(newValue) {
        this.props.history.push("?state="+newValue);
        this.setState({jobList:[]}, ()=>this.refreshData());
    }

    render() {
        const qps = this.getQueryParams();

        return <div>
            <MenuBanner/>
            <h1>Jobs</h1>
            <JobStatusSummary filterClicked={this.setStatusFilter} currentFilterName={qps.state}/>
            <ul className="job-list">
                {
                    this.state.jobList.map(entry=><li className="job-list-entry" key={entry.id}>
                        <div className="job-list-container">
                            <div className="job-list-entry-cell mini"><FontAwesomeIcon icon="tools"/></div>
                            <div className="job-list-entry-cell baseline">
                                <p style={{marginBottom: "0.2em", marginTop: 0}}>{entry.id}</p>
                            </div>
                            <div className="job-list-entry-cell baseline"><JobStatusComponent status={entry.status}/></div>
                            <div className="job-list-entry-cell wide">
                                <p style={{marginBottom: "0.2em", marginTop: 0}}>{this.getTemplateName(entry.templateId)}</p>
                                <TimestampFormatter relative={false} formatString="HH:mm:ss dd Do MMM" value={entry.start_time} prefix="Started " nullValueString="Not started yet"/><br/>
                                <TimestampFormatter relative={false} formatString="HH:mm:ss dd Do MMM" value={entry.end_time} prefix="Completed " nullValueString="Not finished yet"/><br/>
                            </div>
                            <div className="job-list-entry-cell baseline">{
                                entry.associated_bulk ? <a href={"/batch/" + entry.associated_bulk.list}>Bulk list ></a> : <p>(quick job)</p>
                            }</div>
                            <div className="job-list-entry-cell wide">Completed {entry.completed_steps} steps out of {entry.steps.length}<br/><span className="error-text">{entry.error_message}</span></div>
                        </div>
                        <div className="job-list-content-indented">
                            <HidableExpander headerText="Details">
                                <ul className="job-details-list">{
                                    entry.steps.map((step,idx)=>
                                        step===null ? <li className="job-sublist-entry">(invalid data)</li> :
                                        <li className="job-sublist-entry" key={step.id}>{
                                                this.renderJobStepDetails(step,idx)
                                        }</li>)
                                }</ul>
                            </HidableExpander>
                        </div>
                    </li>)
                }
            </ul>
            <Modal open={this.state.showModal} onClose={()=>this.setState({showModal: false})} center>
                <ThumbnailPreview className="thumbnail-large" clickable={false} fileId={this.state.modalThumbnailId}/>
            </Modal>
            <Modal open={this.state.showLogsFor} onClose={()=>this.setState({showLogsFor: null, currentLogContent: null})} center>
                {
                    this.state.currentLogContent ? <pre className="logview">{this.state.currentLogContent}</pre> : <pre>Loading...</pre>
                }
            </Modal>
        </div>
    }
}

export default JobList;