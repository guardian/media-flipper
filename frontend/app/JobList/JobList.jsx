import React from 'react';
import PropTypes from 'prop-types';
import css from "./JobList.css";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import HidableExpander from "../Common/HidableExpander.jsx";
import JobStatusComponent from "./JobStatusComponent.jsx";
import MenuBanner from "../MenuBanner.jsx";
import MediaFileInfo from "./MediaFileInfo.jsx";

class JobList extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            jobList: [],
            loading: false,
            lastError: null,
            jobTemplateLookup: {},
            nextPageCursor: 0
        }
    }

    componentDidMount() {
        this.setState({loading: true}, async ()=>{
            await this.loadJobTemplateLookup();
            await this.loadJobListPage();
        });
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

    async loadJobListPage() {
        const response = await fetch("/api/job");
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
                    <div className="job-list-entry-cell baseline"><JobStatusComponent status={step.jobStepStatus}/></div>
                    <div className="job-list-entry-cell baseline">Format Analysis</div>
                    <div className="job-list-entry-cell baseline">{step.analysisResult && step.analysisResult!=="00000000-0000-0000-0000-000000000000" ? <MediaFileInfo jobId={step.analysisResult}/> : <span/>}</div>
                    <div className="job-list-entry-cell wide">{step.errorMessage}</div>
                    </div>;
            case "thumbnail":
                /*pad out with empty divs so that the columns align*/
                return <div className="job-list-container">
                    <div className="job-list-entry-cell baseline"><FontAwesomeIcon icon="wrench"/>  Step {idx+1}</div>
                    <div className="job-list-entry-cell baseline"><JobStatusComponent status={step.jobStepStatus}/></div>
                    <div className="job-list-entry-cell baseline">Generate thumbnail</div>
                    <div className="job-list-entry-cell baseline"/>
                    <div className="job-list-entry-cell wide">{step.thumbnailResult.errorMessage}</div>
                </div>;
            default:
                return <div className="job-list-container"><div className="job-list-entry-cell wide">Unknown job step type {step.stepType}</div></div>
        }
    }

    render() {
        return <div>
            <MenuBanner/>
            <h1>Jobs</h1>
            <ul className="job-list">
                {
                    this.state.jobList.map(entry=><li className="job-list-entry" key={entry.id}>
                        <div className="job-list-container">
                            <div className="job-list-entry-cell mini"><FontAwesomeIcon icon="tools"/></div>
                            <div className="job-list-entry-cell baseline">{entry.id}</div>
                            <div className="job-list-entry-cell baseline"><JobStatusComponent status={entry.status}/></div>
                            <div className="job-list-entry-cell wide">{this.getTemplateName(entry.templateId)}</div>
                            <div className="job-list-entry-cell wide">Completed {entry.completed_steps} steps out of {entry.steps.length}<br/><span className="error-text">{entry.error_message}</span></div>
                        </div>
                        <div className="job-list-content-indented">
                            <HidableExpander headerText="Details">
                                <ul className="job-details-list">{
                                    entry.steps.map((step,idx)=>
                                        <li className="job-sublist-entry" key={step.id}>{
                                                this.renderJobStepDetails(step,idx)
                                        }</li>)
                                }</ul>
                            </HidableExpander>
                        </div>
                    </li>)
                }
            </ul>
        </div>
    }
}

export default JobList;